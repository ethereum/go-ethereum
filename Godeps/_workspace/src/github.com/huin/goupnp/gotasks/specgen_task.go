// +build gotask

package gotasks

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/scpd"
	"github.com/huin/goutil/codegen"
	"github.com/jingweno/gotask/tasking"
)

var (
	deviceURNPrefix  = "urn:schemas-upnp-org:device:"
	serviceURNPrefix = "urn:schemas-upnp-org:service:"
)

// NAME
//   specgen - generates Go code from the UPnP specification files.
//
// DESCRIPTION
//   The specification is available for download from:
//
// OPTIONS
//   -s, --spec_filename=<upnpresources.zip>
//     Path to the specification file, available from http://upnp.org/resources/upnpresources.zip
//   -o, --out_dir=<output directory>
//     Path to the output directory. This is is where the DCP source files will be placed. Should normally correspond to the directory for github.com/huin/goupnp/dcps
//   --nogofmt
//     Disable passing the output through gofmt. Do this if debugging code output problems and needing to see the generated code prior to being passed through gofmt.
func TaskSpecgen(t *tasking.T) {
	specFilename := t.Flags.String("spec-filename")
	if specFilename == "" {
		specFilename = t.Flags.String("s")
	}
	if specFilename == "" {
		t.Fatal("--spec_filename is required")
	}
	outDir := t.Flags.String("out-dir")
	if outDir == "" {
		outDir = t.Flags.String("o")
	}
	if outDir == "" {
		log.Fatal("--out_dir is required")
	}
	useGofmt := !t.Flags.Bool("nogofmt")

	specArchive, err := openZipfile(specFilename)
	if err != nil {
		t.Fatalf("Error opening spec file: %v", err)
	}
	defer specArchive.Close()

	dcpCol := newDcpsCollection()
	for _, f := range globFiles("standardizeddcps/*/*.zip", specArchive.Reader) {
		dirName := strings.TrimPrefix(f.Name, "standardizeddcps/")
		slashIndex := strings.Index(dirName, "/")
		if slashIndex == -1 {
			// Should not happen.
			t.Logf("Could not find / in %q", dirName)
			return
		}
		dirName = dirName[:slashIndex]

		dcp := dcpCol.dcpForDir(dirName)
		if dcp == nil {
			t.Logf("No alias defined for directory %q: skipping %s\n", dirName, f.Name)
			continue
		} else {
			t.Logf("Alias found for directory %q: processing %s\n", dirName, f.Name)
		}

		dcp.processZipFile(f)
	}

	for _, dcp := range dcpCol.dcpByAlias {
		if err := dcp.writePackage(outDir, useGofmt); err != nil {
			log.Printf("Error writing package %q: %v", dcp.Metadata.Name, err)
		}
	}
}

// DCP contains extra metadata to use when generating DCP source files.
type DCPMetadata struct {
	Name         string // What to name the Go DCP package.
	OfficialName string // Official name for the DCP.
	DocURL       string // Optional - URL for futher documentation about the DCP.
}

var dcpMetadataByDir = map[string]DCPMetadata{
	"Internet Gateway_1": {
		Name:         "internetgateway1",
		OfficialName: "Internet Gateway Device v1",
		DocURL:       "http://upnp.org/specs/gw/UPnP-gw-InternetGatewayDevice-v1-Device.pdf",
	},
	"Internet Gateway_2": {
		Name:         "internetgateway2",
		OfficialName: "Internet Gateway Device v2",
		DocURL:       "http://upnp.org/specs/gw/UPnP-gw-InternetGatewayDevice-v2-Device.pdf",
	},
}

type dcpCollection struct {
	dcpByAlias map[string]*DCP
}

func newDcpsCollection() *dcpCollection {
	c := &dcpCollection{
		dcpByAlias: make(map[string]*DCP),
	}
	for _, metadata := range dcpMetadataByDir {
		c.dcpByAlias[metadata.Name] = newDCP(metadata)
	}
	return c
}

func (c *dcpCollection) dcpForDir(dirName string) *DCP {
	metadata, ok := dcpMetadataByDir[dirName]
	if !ok {
		return nil
	}
	return c.dcpByAlias[metadata.Name]
}

// DCP collects together information about a UPnP Device Control Protocol.
type DCP struct {
	Metadata     DCPMetadata
	DeviceTypes  map[string]*URNParts
	ServiceTypes map[string]*URNParts
	Services     []SCPDWithURN
}

func newDCP(metadata DCPMetadata) *DCP {
	return &DCP{
		Metadata:     metadata,
		DeviceTypes:  make(map[string]*URNParts),
		ServiceTypes: make(map[string]*URNParts),
	}
}

func (dcp *DCP) processZipFile(file *zip.File) {
	archive, err := openChildZip(file)
	if err != nil {
		log.Println("Error reading child zip file:", err)
		return
	}
	for _, deviceFile := range globFiles("*/device/*.xml", archive) {
		dcp.processDeviceFile(deviceFile)
	}
	for _, scpdFile := range globFiles("*/service/*.xml", archive) {
		dcp.processSCPDFile(scpdFile)
	}
}

func (dcp *DCP) processDeviceFile(file *zip.File) {
	var device goupnp.Device
	if err := unmarshalXmlFile(file, &device); err != nil {
		log.Printf("Error decoding device XML from file %q: %v", file.Name, err)
		return
	}
	device.VisitDevices(func(d *goupnp.Device) {
		t := strings.TrimSpace(d.DeviceType)
		if t != "" {
			u, err := extractURNParts(t, deviceURNPrefix)
			if err != nil {
				log.Println(err)
				return
			}
			dcp.DeviceTypes[t] = u
		}
	})
	device.VisitServices(func(s *goupnp.Service) {
		u, err := extractURNParts(s.ServiceType, serviceURNPrefix)
		if err != nil {
			log.Println(err)
			return
		}
		dcp.ServiceTypes[s.ServiceType] = u
	})
}

func (dcp *DCP) writePackage(outDir string, useGofmt bool) error {
	packageDirname := filepath.Join(outDir, dcp.Metadata.Name)
	err := os.MkdirAll(packageDirname, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}
	packageFilename := filepath.Join(packageDirname, dcp.Metadata.Name+".go")
	packageFile, err := os.Create(packageFilename)
	if err != nil {
		return err
	}
	var output io.WriteCloser = packageFile
	if useGofmt {
		if output, err = codegen.NewGofmtWriteCloser(output); err != nil {
			packageFile.Close()
			return err
		}
	}
	if err = packageTmpl.Execute(output, dcp); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func (dcp *DCP) processSCPDFile(file *zip.File) {
	scpd := new(scpd.SCPD)
	if err := unmarshalXmlFile(file, scpd); err != nil {
		log.Printf("Error decoding SCPD XML from file %q: %v", file.Name, err)
		return
	}
	scpd.Clean()
	urnParts, err := urnPartsFromSCPDFilename(file.Name)
	if err != nil {
		log.Printf("Could not recognize SCPD filename %q: %v", file.Name, err)
		return
	}
	dcp.Services = append(dcp.Services, SCPDWithURN{
		URNParts: urnParts,
		SCPD:     scpd,
	})
}

type SCPDWithURN struct {
	*URNParts
	SCPD *scpd.SCPD
}

func (s *SCPDWithURN) WrapArgument(arg scpd.Argument) (*argumentWrapper, error) {
	relVar := s.SCPD.GetStateVariable(arg.RelatedStateVariable)
	if relVar == nil {
		return nil, fmt.Errorf("no such state variable: %q, for argument %q", arg.RelatedStateVariable, arg.Name)
	}
	cnv, ok := typeConvs[relVar.DataType.Name]
	if !ok {
		return nil, fmt.Errorf("unknown data type: %q, for state variable %q, for argument %q", relVar.DataType.Type, arg.RelatedStateVariable, arg.Name)
	}
	return &argumentWrapper{
		Argument: arg,
		relVar:   relVar,
		conv:     cnv,
	}, nil
}

type argumentWrapper struct {
	scpd.Argument
	relVar *scpd.StateVariable
	conv   conv
}

func (arg *argumentWrapper) AsParameter() string {
	return fmt.Sprintf("%s %s", arg.Name, arg.conv.ExtType)
}

func (arg *argumentWrapper) Document() string {
	relVar := arg.relVar
	if rng := relVar.AllowedValueRange; rng != nil {
		var parts []string
		if rng.Minimum != "" {
			parts = append(parts, fmt.Sprintf("minimum=%s", rng.Minimum))
		}
		if rng.Maximum != "" {
			parts = append(parts, fmt.Sprintf("maximum=%s", rng.Maximum))
		}
		if rng.Step != "" {
			parts = append(parts, fmt.Sprintf("step=%s", rng.Step))
		}
		return "allowed value range: " + strings.Join(parts, ", ")
	}
	if len(relVar.AllowedValues) != 0 {
		return "allowed values: " + strings.Join(relVar.AllowedValues, ", ")
	}
	return ""
}

func (arg *argumentWrapper) Marshal() string {
	return fmt.Sprintf("soap.Marshal%s(%s)", arg.conv.FuncSuffix, arg.Name)
}

func (arg *argumentWrapper) Unmarshal(objVar string) string {
	return fmt.Sprintf("soap.Unmarshal%s(%s.%s)", arg.conv.FuncSuffix, objVar, arg.Name)
}

type conv struct {
	FuncSuffix string
	ExtType    string
}

// typeConvs maps from a SOAP type (e.g "fixed.14.4") to the function name
// suffix inside the soap module (e.g "Fixed14_4") and the Go type.
var typeConvs = map[string]conv{
	"ui1":         conv{"Ui1", "uint8"},
	"ui2":         conv{"Ui2", "uint16"},
	"ui4":         conv{"Ui4", "uint32"},
	"i1":          conv{"I1", "int8"},
	"i2":          conv{"I2", "int16"},
	"i4":          conv{"I4", "int32"},
	"int":         conv{"Int", "int64"},
	"r4":          conv{"R4", "float32"},
	"r8":          conv{"R8", "float64"},
	"number":      conv{"R8", "float64"}, // Alias for r8.
	"fixed.14.4":  conv{"Fixed14_4", "float64"},
	"float":       conv{"R8", "float64"},
	"char":        conv{"Char", "rune"},
	"string":      conv{"String", "string"},
	"date":        conv{"Date", "time.Time"},
	"dateTime":    conv{"DateTime", "time.Time"},
	"dateTime.tz": conv{"DateTimeTz", "time.Time"},
	"time":        conv{"TimeOfDay", "soap.TimeOfDay"},
	"time.tz":     conv{"TimeOfDayTz", "soap.TimeOfDay"},
	"boolean":     conv{"Boolean", "bool"},
	"bin.base64":  conv{"BinBase64", "[]byte"},
	"bin.hex":     conv{"BinHex", "[]byte"},
}

type closeableZipReader struct {
	io.Closer
	*zip.Reader
}

func openZipfile(filename string) (*closeableZipReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	archive, err := zip.NewReader(file, fi.Size())
	if err != nil {
		return nil, err
	}
	return &closeableZipReader{
		Closer: file,
		Reader: archive,
	}, nil
}

// openChildZip opens a zip file within another zip file.
func openChildZip(file *zip.File) (*zip.Reader, error) {
	zipFile, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer zipFile.Close()

	zipBytes, err := ioutil.ReadAll(zipFile)
	if err != nil {
		return nil, err
	}

	return zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
}

func globFiles(pattern string, archive *zip.Reader) []*zip.File {
	var files []*zip.File
	for _, f := range archive.File {
		if matched, err := path.Match(pattern, f.Name); err != nil {
			// This shouldn't happen - all patterns are hard-coded, errors in them
			// are a programming error.
			panic(err)
		} else if matched {
			files = append(files, f)
		}
	}
	return files
}

func unmarshalXmlFile(file *zip.File, data interface{}) error {
	r, err := file.Open()
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(r)
	r.Close()
	return decoder.Decode(data)
}

type URNParts struct {
	URN     string
	Name    string
	Version string
}

func (u *URNParts) Const() string {
	return fmt.Sprintf("URN_%s_%s", u.Name, u.Version)
}

// extractURNParts extracts the name and version from a URN string.
func extractURNParts(urn, expectedPrefix string) (*URNParts, error) {
	if !strings.HasPrefix(urn, expectedPrefix) {
		return nil, fmt.Errorf("%q does not have expected prefix %q", urn, expectedPrefix)
	}
	parts := strings.SplitN(strings.TrimPrefix(urn, expectedPrefix), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%q does not have a name and version", urn)
	}
	name, version := parts[0], parts[1]
	return &URNParts{urn, name, version}, nil
}

var scpdFilenameRe = regexp.MustCompile(
	`.*/([a-zA-Z0-9]+)([0-9]+)\.xml`)

func urnPartsFromSCPDFilename(filename string) (*URNParts, error) {
	parts := scpdFilenameRe.FindStringSubmatch(filename)
	if len(parts) != 3 {
		return nil, fmt.Errorf("SCPD filename %q does not have expected number of parts", filename)
	}
	name, version := parts[1], parts[2]
	return &URNParts{
		URN:     serviceURNPrefix + name + ":" + version,
		Name:    name,
		Version: version,
	}, nil
}

var packageTmpl = template.Must(template.New("package").Parse(`{{$name := .Metadata.Name}}
// Client for UPnP Device Control Protocol {{.Metadata.OfficialName}}.
// {{if .Metadata.DocURL}}
// This DCP is documented in detail at: {{.Metadata.DocURL}}{{end}}
//
// Typically, use one of the New* functions to discover services on the local
// network.
package {{$name}}

// Generated file - do not edit by hand. See README.md


import (
	"time"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
)

// Hack to avoid Go complaining if time isn't used.
var _ time.Time

// Device URNs:
const ({{range .DeviceTypes}}
	{{.Const}} = "{{.URN}}"{{end}}
)

// Service URNs:
const ({{range .ServiceTypes}}
	{{.Const}} = "{{.URN}}"{{end}}
)

{{range .Services}}
{{$srv := .}}
{{$srvIdent := printf "%s%s" .Name .Version}}

// {{$srvIdent}} is a client for UPnP SOAP service with URN "{{.URN}}". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type {{$srvIdent}} struct {
	goupnp.ServiceClient
}

// New{{$srvIdent}}Clients discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func New{{$srvIdent}}Clients() (clients []*{{$srvIdent}}, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClients({{$srv.Const}}); err != nil {
		return
	}
	clients = make([]*{{$srvIdent}}, len(genericClients))
	for i := range genericClients {
		clients[i] = &{{$srvIdent}}{genericClients[i]}
	}
	return
}

{{range .SCPD.Actions}}{{/* loops over *SCPDWithURN values */}}

{{$inargs := .InputArguments}}{{$outargs := .OutputArguments}}
// {{if $inargs}}Arguments:{{range $inargs}}{{$argWrap := $srv.WrapArgument .}}
//
// * {{.Name}}: {{$argWrap.Document}}{{end}}{{end}}
//
// {{if $outargs}}Return values:{{range $outargs}}{{$argWrap := $srv.WrapArgument .}}
//
// * {{.Name}}: {{$argWrap.Document}}{{end}}{{end}}
func (client *{{$srvIdent}}) {{.Name}}({{range $inargs}}{{/*
*/}}{{$argWrap := $srv.WrapArgument .}}{{$argWrap.AsParameter}}, {{end}}{{/*
*/}}) ({{range $outargs}}{{/*
*/}}{{$argWrap := $srv.WrapArgument .}}{{$argWrap.AsParameter}}, {{end}} err error) {
	// Request structure.
	request := {{if $inargs}}&{{template "argstruct" $inargs}}{{"{}"}}{{else}}{{"interface{}(nil)"}}{{end}}
	// BEGIN Marshal arguments into request.
{{range $inargs}}{{$argWrap := $srv.WrapArgument .}}
	if request.{{.Name}}, err = {{$argWrap.Marshal}}; err != nil {
		return
	}{{end}}
	// END Marshal arguments into request.

	// Response structure.
	response := {{if $outargs}}&{{template "argstruct" $outargs}}{{"{}"}}{{else}}{{"interface{}(nil)"}}{{end}}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformAction({{$srv.URNParts.Const}}, "{{.Name}}", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.
{{range $outargs}}{{$argWrap := $srv.WrapArgument .}}
	if {{.Name}}, err = {{$argWrap.Unmarshal "response"}}; err != nil {
		return
	}{{end}}
	// END Unmarshal arguments from response.
	return
}
{{end}}{{/* range .SCPD.Actions */}}
{{end}}{{/* range .Services */}}

{{define "argstruct"}}struct {{"{"}}{{range .}}
{{.Name}} string
{{end}}{{"}"}}{{end}}
`))
