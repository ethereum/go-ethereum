# globalconf

[![Build Status](https://travis-ci.org/rakyll/globalconf.png?branch=master)](https://travis-ci.org/rakyll/globalconf)

Effortlessly persist/retrieve flags of your Golang programs. If you need global configuration instead of requiring user always to set command line flags, you are looking at the right package. `globalconf` allows your users to not only provide flags, but config files and environment variables as well.

## Usage

~~~ go
import "github.com/rakyll/globalconf"
~~~

### Loading a config file

By default, globalconf provides you a config file under `~/.config/<yourappname>/config.ini`.

~~~ go
globalconf.New("appname") // loads from ~/.config/<appname>/config.ini
~~~

If you don't prefer the default location you can load from a specified path as well.

~~~ go
globalconf.NewWithOptions(&globalconf.Options{
	Filename: "/path/to/config/file",
})
~~~

You may like to override configuration with env variables. See "Environment variables" header to see how to it works.

~~~ go
globalconf.NewWithOptions(&globalconf.Options{
	Filename:  "/path/to/config/file",
	EnvPrefix: "APPCONF_",
})
~~~

### Parsing flag values

`globalconf` populates flags with data in the config file if they are not already set.

~~~ go
var (
	flagName    = flag.String("name", "", "Name of the person.")
	flagAddress = flag.String("addr", "", "Address of the person.")
)
~~~
	
Assume the configuration file to be loaded contains the following lines.

	name = Burcu
	addr = Brandschenkestrasse 110, 8002

And your program is being started, `$ myapp -name=Jane`
~~~ go
conf, err := globalconf.New("myapp")
conf.ParseAll()
~~~

`*flagName` is going to be equal to `Jane`, whereas `*flagAddress` is `Brandschenkestrasse 110, 8002`, what is provided in the configuration file.

### Custom flag sets

Custom flagsets are supported, but required registration before parse is done. The default flagset `flag.CommandLine` is automatically registered.

~~~ go
globalconf.Register("termopts", termOptsFlagSet)
conf.ParseAll() // parses command line and all registered flag sets
~~~

Custom flagset values should be provided in their own segment. Getting back to the sample ini config file, termopts values will have their own segment.

	name = Burcu
	addr = Brandschenkestrasse 110, 8002
	
	[termopts]
	color = true
	background = ff0000

### Environment variables

If an EnvPrefix is provided, environment variables will take precedence over values in the configuration file.
Set the `EnvPrefix` option when calling `globalconf.NewWithOptions`.
An `EnvPrefix` will only be used if it is a non-empty string.
Command line flags will override the environment variables.

~~~ go
opts := globalconf.Options{
	EnvPrefix: "MYAPP_",
	Filename:  "/path/to/config",
}
conf, err := globalconf.NewWithOptions(&opts)
conf.ParseAll()
~~~

With environment variables:

	APPCONF_NAME = Burcu

and configuration:

	name = Jane
	addr = Brandschenkestrasse 110, 8002

`name` will be set to "burcu" and `addr` will be set to "Brandschenkestrasse 110, 8002".

### Modifying stored flags

Modifications are persisted as long as you set a new flag and your GlobalConf
object was configured with a filename.

~~~ go
f := &flag.Flag{Name: "name", Value: val}
conf.Set("", f) // if you are modifying a command line flag

f := &flag.Flag{Name: "color", Value: val}
conf.Set("termopts", color) // if you are modifying a custom flag set flag
~~~

### Deleting stored flags

Like Set, Deletions are persisted as long as you delete a flag's value and your
GlobalConf object was configured with a filename.

~~~ go
conf.Delete("", "name") // removes command line flag "name"s value from config
conf.Delete("termopts", "color") // removes "color"s value from the custom flag set
~~~

## License

Copyright 2014 Google Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. ![Analytics](https://ga-beacon.appspot.com/UA-46881978-1/globalconf?pixel)
