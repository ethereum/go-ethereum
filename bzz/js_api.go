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

func (self *JSApi) get(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		fmt.Println("requires 1 argument: bzz.get(path)")
		return otto.UndefinedValue()
	}

	var err error
	var bzzpath, res string
	bzzpath, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	res, err = self.api.Get(bzzpath)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	v, _ := call.Otto.ToValue(res)
	return v
}

func (self *JSApi) put(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 && len(call.ArgumentList) != 4 {
		fmt.Println("requires 2 or 4 arguments: bzz.put(content, content-type[, address, domain])")
		return otto.UndefinedValue()
	}

	var err error
	var res, content, contentType, address, domain string
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
	if len(call.ArgumentList) > 2 {
		address, err = call.Argument(2).ToString()
		if err != nil {
			fmt.Println(err)
			return otto.UndefinedValue()
		}
		domain, err = call.Argument(3).ToString()
		if err != nil {
			fmt.Println(err)
			return otto.UndefinedValue()
		}
	}

	res, err = self.api.Put(content, contentType, address, domain)
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
	if len(call.ArgumentList) != 1 && len(call.ArgumentList) != 3 {
		fmt.Println("requires 1 or 3 arguments: bzz.put(localpath[, address, domain])")
		return otto.UndefinedValue()
	}

	var err error
	var localpath, address, domain, res string
	localpath, err = call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	if len(call.ArgumentList) > 1 {
		address, err = call.Argument(1).ToString()
		if err != nil {
			fmt.Println(err)
			return otto.UndefinedValue()
		}
		domain, err = call.Argument(2).ToString()
		if err != nil {
			fmt.Println(err)
			return otto.UndefinedValue()
		}
	}

	res, err = self.api.Upload(localpath, address, domain)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	v, _ := call.Otto.ToValue(res)
	return v
}

// http.PostForm("http://example.com/form",
//   url.Values{"key": {"Value"}, "id": {"123"}})
