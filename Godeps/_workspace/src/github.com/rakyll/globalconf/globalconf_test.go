package globalconf

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"
)

const envTestPrefix = "CONFTEST_"

func TestNewWithOptionsNoFilename(t *testing.T) {
	opts := Options{EnvPrefix: envTestPrefix}

	os.Setenv(envTestPrefix+"D", "EnvD")

	flagD := flag.String("d", "default", "")
	flagE := flag.Bool("e", true, "")

	conf, err := NewWithOptions(&opts)
	if err != nil {
		t.Fatal(err)
	}
	conf.ParseAll()

	if *flagD != "EnvD" {
		t.Errorf("flagD found %v, expected 'EnvD'", *flagD)
	}
	if !*flagE {
		t.Errorf("flagE found %v, expected true", *flagE)
	}
}

func TestParse_Global(t *testing.T) {
	resetForTesting("")

	os.Setenv(envTestPrefix+"D", "EnvD")
	os.Setenv(envTestPrefix+"E", "true")
	os.Setenv(envTestPrefix+"F", "5.5")

	flagA := flag.Bool("a", false, "")
	flagB := flag.Float64("b", 0.0, "")
	flagC := flag.String("c", "", "")

	flagD := flag.String("d", "", "")
	flagE := flag.Bool("e", false, "")
	flagF := flag.Float64("f", 0.0, "")

	parse(t, "./testdata/global.ini", envTestPrefix)
	if !*flagA {
		t.Errorf("flagA found %v, expected true", *flagA)
	}
	if *flagB != 5.6 {
		t.Errorf("flagB found %v, expected 5.6", *flagB)
	}
	if *flagC != "Hello world" {
		t.Errorf("flagC found %v, expected 'Hello world'", *flagC)
	}
	if *flagD != "EnvD" {
		t.Errorf("flagD found %v, expected 'EnvD'", *flagD)
	}
	if !*flagE {
		t.Errorf("flagE found %v, expected true", *flagE)
	}
	if *flagF != 5.5 {
		t.Errorf("flagF found %v, expected 5.5", *flagF)
	}
}

func TestParse_DashConversion(t *testing.T) {
	resetForTesting("")

	flagFooBar := flag.String("foo-bar", "", "")
	os.Setenv("PREFIX_FOO_BAR", "baz")

	opts := Options{EnvPrefix: "PREFIX_"}
	conf, err := NewWithOptions(&opts)
	if err != nil {
		t.Fatal(err)
	}
	conf.ParseAll()

	if *flagFooBar != "baz" {
		t.Errorf("flagFooBar found %v, expected 5.5", *flagFooBar)
	}
}

func TestParse_GlobalWithDottedFlagname(t *testing.T) {
	resetForTesting("")
	os.Setenv(envTestPrefix+"SOME_VALUE", "some-value")
	flagSomeValue := flag.String("some.value", "", "")

	parse(t, "./testdata/global.ini", envTestPrefix)
	if *flagSomeValue != "some-value" {
		t.Errorf("flagSomeValue found %v, some-value expected", *flagSomeValue)
	}
}

func TestParse_GlobalOverwrite(t *testing.T) {
	resetForTesting("-b=7.6")
	flagB := flag.Float64("b", 0.0, "")

	parse(t, "./testdata/global.ini", "")
	if *flagB != 7.6 {
		t.Errorf("flagB found %v, expected 7.6", *flagB)
	}
}

func TestParse_Custom(t *testing.T) {
	resetForTesting("")

	os.Setenv(envTestPrefix+"CUSTOM_E", "Hello Env")

	flagB := flag.Float64("b", 5.0, "")

	name := "custom"
	custom := flag.NewFlagSet(name, flag.ExitOnError)
	flagD := custom.String("d", "dd", "")
	flagE := custom.String("e", "ee", "")

	Register(name, custom)
	parse(t, "./testdata/custom.ini", envTestPrefix)
	if *flagB != 5.0 {
		t.Errorf("flagB found %v, expected 5.0", *flagB)
	}
	if *flagD != "Hello d" {
		t.Errorf("flagD found %v, expected 'Hello d'", *flagD)
	}
	if *flagE != "Hello Env" {
		t.Errorf("flagE found %v, expected 'Hello Env'", *flagE)
	}
}

func TestParse_CustomOverwrite(t *testing.T) {
	resetForTesting("-b=6")
	flagB := flag.Float64("b", 5.0, "")

	name := "custom"
	custom := flag.NewFlagSet(name, flag.ExitOnError)
	flagD := custom.String("d", "dd", "")

	Register(name, custom)
	parse(t, "./testdata/custom.ini", "")
	if *flagB != 6.0 {
		t.Errorf("flagB found %v, expected 6.0", *flagB)
	}
	if *flagD != "Hello d" {
		t.Errorf("flagD found %v, expected 'Hello d'", *flagD)
	}
}

func TestParse_GlobalAndCustom(t *testing.T) {
	resetForTesting("")
	flagA := flag.Bool("a", false, "")
	flagB := flag.Float64("b", 0.0, "")
	flagC := flag.String("c", "", "")

	name := "custom"
	custom := flag.NewFlagSet(name, flag.ExitOnError)
	flagD := custom.String("d", "", "")

	Register(name, custom)
	parse(t, "./testdata/globalandcustom.ini", "")
	if !*flagA {
		t.Errorf("flagA found %v, expected true", *flagA)
	}
	if *flagB != 5.6 {
		t.Errorf("flagB found %v, expected 5.6", *flagB)
	}
	if *flagC != "Hello world" {
		t.Errorf("flagC found %v, expected 'Hello world'", *flagC)
	}
	if *flagD != "Hello d" {
		t.Errorf("flagD found %v, expected 'Hello d'", *flagD)
	}
}

func TestParse_GlobalAndCustomOverwrite(t *testing.T) {
	resetForTesting("-a=true", "-b=5", "-c=Hello")
	flagA := flag.Bool("a", false, "")
	flagB := flag.Float64("b", 0.0, "")
	flagC := flag.String("c", "", "")

	name := "custom"
	custom := flag.NewFlagSet(name, flag.ExitOnError)
	flagD := custom.String("d", "", "")

	Register(name, custom)
	parse(t, "./testdata/globalandcustom.ini", "")
	if !*flagA {
		t.Errorf("flagA found %v, expected true", *flagA)
	}
	if *flagB != 5.0 {
		t.Errorf("flagB found %v, expected 5.0", *flagB)
	}
	if *flagC != "Hello" {
		t.Errorf("flagC found %v, expected 'Hello'", *flagC)
	}
	if *flagD != "Hello d" {
		t.Errorf("flagD found %v, expected 'Hello d'", *flagD)
	}
}

func TestSet(t *testing.T) {
	resetForTesting()
	file, _ := ioutil.TempFile("", "")
	conf := parse(t, file.Name(), "")
	conf.Set("", &flag.Flag{Name: "a", Value: newFlagValue("test")})

	flagA := flag.String("a", "", "")
	parse(t, file.Name(), "")
	if *flagA != "test" {
		t.Errorf("flagA found %v, expected 'test'", *flagA)
	}
}

func TestDelete(t *testing.T) {
	resetForTesting()
	file, _ := ioutil.TempFile("", "")
	conf := parse(t, file.Name(), "")
	conf.Set("", &flag.Flag{Name: "a", Value: newFlagValue("test")})
	conf.Delete("", "a")

	flagA := flag.String("a", "", "")
	parse(t, file.Name(), "")
	if *flagA != "" {
		t.Errorf("flagNewA found %v, expected ''", *flagA)
	}
}

func parse(t *testing.T, filename, envPrefix string) *GlobalConf {
	opts := Options{
		Filename:  filename,
		EnvPrefix: envPrefix,
	}
	conf, err := NewWithOptions(&opts)
	if err != nil {
		t.Error(err)
	}
	conf.ParseAll()
	return conf
}

// Resets os.Args and the default flag set.
func resetForTesting(args ...string) {
	os.Clearenv()

	os.Args = append([]string{"cmd"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

type flagValue struct {
	str string
}

func (f *flagValue) String() string {
	return f.str
}

func (f *flagValue) Set(value string) error {
	f.str = value
	return nil
}

func newFlagValue(val string) *flagValue {
	return &flagValue{str: val}
}
