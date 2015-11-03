// +build gotask

package gotasks

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
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

// DCP contains extra metadata to use when generating DCP source files.
type DCPMetadata struct {
	Name         string // What to name the Go DCP package.
	OfficialName string // Official name for the DCP.
	DocURL       string // Optional - URL for futher documentation about the DCP.
	XMLSpecURL   string // Where to download the XML spec from.
	// Any special-case functions to run against the DCP before writing it out.
	Hacks []DCPHackFn
}

var dcpMetadata = []DCPMetadata{
	{
		Name:         "internetgateway1",
		OfficialName: "Internet Gateway Device v1",
		DocURL:       "http://upnp.org/specs/gw/UPnP-gw-InternetGatewayDevice-v1-Device.pdf",
		XMLSpecURL:   "http://upnp.org/specs/gw/UPnP-gw-IGD-TestFiles-20010921.zip",
	},
	{
		Name:         "internetgateway2",
		OfficialName: "Internet Gateway Device v2",
		DocURL:       "http://upnp.org/specs/gw/UPnP-gw-InternetGatewayDevice-v2-Device.pdf",
		XMLSpecURL:   "http://upnp.org/specs/gw/UPnP-gw-IGD-Testfiles-20110224.zip",
		Hacks: []DCPHackFn{
			func(dcp *DCP) error {
				missingURN := "urn:schemas-upnp-org:service:WANIPv6FirewallControl:1"
				if _, ok := dcp.ServiceTypes[missingURN]; ok {
					return nil
				}
				urnParts, err := extractURNParts(missingURN, serviceURNPrefix)
				if err != nil {
					return err
				}
				dcp.ServiceTypes[missingURN] = urnParts
				return nil
			},
		},
	},
	{
		Name:         "av1",
		OfficialName: "MediaServer v1 and MediaRenderer v1",
		DocURL:       "http://upnp.org/specs/av/av1/",
		XMLSpecURL:   "http://upnp.org/specs/av/UPnP-av-TestFiles-20070927.zip",
	},
}

type DCPHackFn func(*DCP) error

// NAME
//   specgen - generates Go code from the UPnP specification files.
//
// DESCRIPTION
//   The specification is available for download from:
//
// OPTIONS
//   -s, --specs_dir=<spec directory>
//     Path to the specification storage directory. This is used to find (and download if not present) the specification ZIP files. Defaults to 'specs'
//   -o, --out_dir=<output directory>
//     Path to the output directory. This is is where the DCP source files will be placed. Should normally correspond to the directory for github.com/huin/goupnp/dcps. Defaults to '../dcps'
//   --nogofmt
//     Disable passing the output through gofmt. Do this if debugging code output problems and needing to see the generated code prior to being passed through gofmt.
func TaskSpecgen(t *tasking.T) {
	specsDir := fallbackStrValue("specs", t.Flags.String("specs_dir"), t.Flags.String("s"))
	if err := os.MkdirAll(specsDir, os.ModePerm); err != nil {
		t.Fatalf("Could not create specs-dir %q: %v\n", specsDir, err)
	}
	outDir := fallbackStrValue("../dcps", t.Flags.String("out_dir"), t.Flags.String("o"))
	useGofmt := !t.Flags.Bool("nogofmt")

NEXT_DCP:
	for _, d := range dcpMetadata {
		specFilename := filepath.Join(specsDir, d.Name+".zip")
		err := acquireFile(specFilename, d.XMLSpecURL)
		if err != nil {
			t.Logf("Could not acquire spec for %s, skipping: %v\n", d.Name, err)
			continue NEXT_DCP
		}
		dcp := newDCP(d)
		if err := dcp.processZipFile(specFilename); err != nil {
			log.Printf("Error processing spec for %s in file %q: %v", d.Name, specFilename, err)
			continue NEXT_DCP
		}
		for i, hack := range d.Hacks {
			if err := hack(dcp); err != nil {
				log.Printf("Error with Hack[%d] for %s: %v", i, d.Name, err)
				continue NEXT_DCP
			}
		}
		dcp.writePackage(outDir, useGofmt)
		if err := dcp.writePackage(outDir, useGofmt); err != nil {
			log.Printf("Error writing package %q: %v", dcp.Metadata.Name, err)
			continue NEXT_DCP
		}
	}
}

func fallbackStrValue(defaultValue string, values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return defaultValue
}

func acquireFile(specFilename string, xmlSpecURL string) error {
	if f, err := os.Open(specFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		f.Close()
		return nil
	}

	resp, err := http.Get(xmlSpecURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download spec %q from %q: ",
			specFilename, xmlSpecURL, resp.Status)
	}

	tmpFilename := specFilename + ".download"
	w, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return os.Rename(tmpFilename, specFilename)
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

func (dcp *DCP) processZipFile(filename string) error {
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("error reading zip file %q: %v", filename, err)
	}
	defer archive.Close()
	for _, deviceFile := range globFiles("*/device/*.xml", archive) {
		if err := dcp.processDeviceFile(deviceFile); err != nil {
			return err
		}
	}
	for _, scpdFile := range globFiles("*/service/*.xml", archive) {
		if err := dcp.processSCPDFile(scpdFile); err != nil {
			return err
		}
	}
	return nil
}

func (dcp *DCP) processDeviceFile(file *zip.File) error {
	var device goupnp.Device
	if err := unmarshalXmlFile(file, &device); err != nil {
		return fmt.Errorf("error decoding device XML from file %q: %v", file.Name, err)
	}
	var mainErr error
	device.VisitDevices(func(d *goupnp.Device) {
		t := strings.TrimSpace(d.DeviceType)
		if t != "" {
			u, err := extractURNParts(t, deviceURNPrefix)
			if err != nil {
				mainErr = err
			}
			dcp.DeviceTypes[t] = u
		}
	})
	device.VisitServices(func(s *goupnp.Service) {
		u, err := extractURNParts(s.ServiceType, serviceURNPrefix)
		if err != nil {
			mainErr = err
		}
		dcp.ServiceTypes[s.ServiceType] = u
	})
	return mainErr
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

func (dcp *DCP) processSCPDFile(file *zip.File) error {
	scpd := new(scpd.SCPD)
	if err := unmarshalXmlFile(file, scpd); err != nil {
		return fmt.Errorf("error decoding SCPD XML from file %q: %v", file.Name, err)
	}
	scpd.Clean()
	urnParts, err := urnPartsFromSCPDFilename(file.Name)
	if err != nil {
		return fmt.Errorf("could not recognize SCPD filename %q: %v", file.Name, err)
	}
	dcp.Services = append(dcp.Services, SCPDWithURN{
		URNParts: urnParts,
		SCPD:     scpd,
	})
	return nil
}

type SCPDWithURN struct {
	*URNParts
	SCPD *scpd.SCPD
}

func (s *SCPDWithURN) WrapArguments(args []*scpd.Argument) (argumentWrapperList, error) {
	wrappedArgs := make(argumentWrapperList, len(args))
	for i, arg := range args {
		wa, err := s.wrapArgument(arg)
		if err != nil {
			return nil, err
		}
		wrappedArgs[i] = wa
	}
	return wrappedArgs, nil
}

func (s *SCPDWithURN) wrapArgument(arg *scpd.Argument) (*argumentWrapper, error) {
	relVar := s.SCPD.GetStateVariable(arg.RelatedStateVariable)
	if relVar == nil {
		return nil, fmt.Errorf("no such state variable: %q, for argument %q", arg.RelatedStateVariable, arg.Name)
	}
	cnv, ok := typeConvs[relVar.DataType.Name]
	if !ok {
		return nil, fmt.Errorf("unknown data type: %q, for state variable %q, for argument %q", relVar.DataType.Type, arg.RelatedStateVariable, arg.Name)
	}
	return &argumentWrapper{
		Argument: *arg,
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

func (arg *argumentWrapper) HasDoc() bool {
	rng := arg.relVar.AllowedValueRange
	return ((rng != nil && (rng.Minimum != "" || rng.Maximum != "" || rng.Step != "")) ||
		len(arg.relVar.AllowedValues) > 0)
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

type argumentWrapperList []*argumentWrapper

func (args argumentWrapperList) HasDoc() bool {
	for _, arg := range args {
		if arg.HasDoc() {
			return true
		}
	}
	return false
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
	"uri":         conv{"URI", "*url.URL"},
}

func globFiles(pattern string, archive *zip.ReadCloser) []*zip.File {
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
// Typically, use one of the New* functions to create clients for services.
package {{$name}}

// Generated file - do not edit by hand. See README.md


import (
	"net/url"
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
	clients = new{{$srvIdent}}ClientsFromGenericClients(genericClients)
	return
}

// New{{$srvIdent}}ClientsByURL discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func New{{$srvIdent}}ClientsByURL(loc *url.URL) ([]*{{$srvIdent}}, error) {
	genericClients, err := goupnp.NewServiceClientsByURL(loc, {{$srv.Const}})
	if err != nil {
		return nil, err
	}
	return new{{$srvIdent}}ClientsFromGenericClients(genericClients), nil
}

// New{{$srvIdent}}ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func New{{$srvIdent}}ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*{{$srvIdent}}, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, {{$srv.Const}})
	if err != nil {
		return nil, err
	}
	return new{{$srvIdent}}ClientsFromGenericClients(genericClients), nil
}

func new{{$srvIdent}}ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*{{$srvIdent}} {
	clients := make([]*{{$srvIdent}}, len(genericClients))
	for i := range genericClients {
		clients[i] = &{{$srvIdent}}{genericClients[i]}
	}
	return clients
}

{{range .SCPD.Actions}}{{/* loops over *SCPDWithURN values */}}

{{$winargs := $srv.WrapArguments .InputArguments}}
{{$woutargs := $srv.WrapArguments .OutputArguments}}
{{if $winargs.HasDoc}}
//
// Arguments:{{range $winargs}}{{if .HasDoc}}
//
// * {{.Name}}: {{.Document}}{{end}}{{end}}{{end}}
{{if $woutargs.HasDoc}}
//
// Return values:{{range $woutargs}}{{if .HasDoc}}
//
// * {{.Name}}: {{.Document}}{{end}}{{end}}{{end}}
func (client *{{$srvIdent}}) {{.Name}}({{range $winargs}}{{/*
*/}}{{.AsParameter}}, {{end}}{{/*
*/}}) ({{range $woutargs}}{{/*
*/}}{{.AsParameter}}, {{end}} err error) {
	// Request structure.
	request := {{if $winargs}}&{{template "argstruct" $winargs}}{{"{}"}}{{else}}{{"interface{}(nil)"}}{{end}}
	// BEGIN Marshal arguments into request.
{{range $winargs}}
	if request.{{.Name}}, err = {{.Marshal}}; err != nil {
		return
	}{{end}}
	// END Marshal arguments into request.

	// Response structure.
	response := {{if $woutargs}}&{{template "argstruct" $woutargs}}{{"{}"}}{{else}}{{"interface{}(nil)"}}{{end}}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformAction({{$srv.URNParts.Const}}, "{{.Name}}", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.
{{range $woutargs}}
	if {{.Name}}, err = {{.Unmarshal "response"}}; err != nil {
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
