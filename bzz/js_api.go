package bzz

import (
	"fmt"
	// "net/http"

	"github.com/ethereum/go-ethereum/jsre"
	"github.com/robertkrimen/otto"
)

func NewJSApi(vm *jsre.JSRE, api *Api) (jsapi *JSApi) {
	jsapi = &JSApi{
		vm:  vm,
		api: api,
	}
	vm.Set("bzz", struct{}{})
	t, _ := vm.Get("bzz")
	o := t.Object()
	o.Set("register", jsapi.register)
	o.Set("download", jsapi.download)
	o.Set("upload", jsapi.upload)
	o.Set("get", jsapi.get)
	o.Set("put", jsapi.put)

	return
}

type JSApi struct {
	vm  *jsre.JSRE
	api *Api
}

func (self *JSApi) register(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		fmt.Println("requires 3 arguments: bzz.register(address, contenthash, domain)")
		return otto.UndefinedValue()
	}

	var err error
	var sender, contenthash, domain string
	sender, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	contenthash, err = call.Argument(1).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	domain, err = call.Argument(2).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	err = self.api.Register(sender, contenthash, domain)
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *JSApi) get(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		fmt.Println("requires 1 argument: bzz.get(path)")
		return otto.UndefinedValue()
	}

	var err error
	var bzzpath string
	bzzpath, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	var content []byte
	var mimeType string
	var status, size int
	content, mimeType, status, size, err = self.api.Get(bzzpath)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	obj := map[string]string{
		"content":     string(content),
		"contentType": mimeType,
		"status":      fmt.Sprintf("%v", status),
		"size":        fmt.Sprintf("%v", size),
	}

	v, _ := call.Otto.ToValue(obj)
	return v
}

func (self *JSApi) put(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		fmt.Println("requires 2 arguments: bzz.put(content, content-type)")
		return otto.UndefinedValue()
	}

	var err error
	var res, content, contentType string
	content, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	contentType, err = call.Argument(1).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	res, err = self.api.Put(content, contentType)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	v, _ := call.Otto.ToValue(res)
	return v
}

func (self *JSApi) download(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		fmt.Println("requires 2 arguments: bzz.download(bzzpath, localpath)")
		return otto.UndefinedValue()
	}

	var err error
	var bzzpath, localpath, res string
	bzzpath, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	localpath, err = call.Argument(1).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	res, err = self.api.Download(bzzpath, localpath)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	v, _ := call.Otto.ToValue(res)
	return v
}

func (self *JSApi) upload(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		fmt.Println("requires 1 arguments: bzz.put(localpath)")
		return otto.UndefinedValue()
	}

	var err error
	var localpath, res string
	localpath, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	res, err = self.api.Upload(localpath)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	v, _ := call.Otto.ToValue(res)
	return v
}

// http.PostForm("http://example.com/form",
//   url.Values{"key": {"Value"}, "id": {"123"}})
