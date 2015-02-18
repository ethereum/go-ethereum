package soap

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

type capturingRoundTripper struct {
	err         error
	resp        *http.Response
	capturedReq *http.Request
}

func (rt *capturingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.capturedReq = req
	return rt.resp, rt.err
}

func TestActionInputs(t *testing.T) {
	url, err := url.Parse("http://example.com/soap")
	if err != nil {
		t.Fatal(err)
	}
	rt := &capturingRoundTripper{
		err: nil,
		resp: &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(bytes.NewBufferString(`
				<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
					<s:Body>
						<u:myactionResponse xmlns:u="mynamespace">
							<A>valueA</A>
							<B>valueB</B>
						</u:myactionResponse>
					</s:Body>
				</s:Envelope>
			`)),
		},
	}
	client := SOAPClient{
		EndpointURL: *url,
		HTTPClient: http.Client{
			Transport: rt,
		},
	}

	type In struct {
		Foo string
		Bar string `soap:"bar"`
	}
	type Out struct {
		A string
		B string
	}
	in := In{"foo", "bar"}
	gotOut := Out{}
	err = client.PerformAction("mynamespace", "myaction", &in, &gotOut)
	if err != nil {
		t.Fatal(err)
	}

	wantBody := (soapPrefix +
		`<u:myaction xmlns:u="mynamespace">` +
		`<Foo>foo</Foo>` +
		`<bar>bar</bar>` +
		`</u:myaction>` +
		soapSuffix)
	body, err := ioutil.ReadAll(rt.capturedReq.Body)
	if err != nil {
		t.Fatal(err)
	}
	gotBody := string(body)
	if wantBody != gotBody {
		t.Errorf("Bad request body\nwant: %q\n got: %q", wantBody, gotBody)
	}

	wantOut := Out{"valueA", "valueB"}
	if !reflect.DeepEqual(wantOut, gotOut) {
		t.Errorf("Bad output\nwant: %+v\n got: %+v", wantOut, gotOut)
	}
}
