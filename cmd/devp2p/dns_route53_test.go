package main

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/route53"
)

// This test checks that computeChanges/splitChanges create DNS changes in
// leaf-added -> root-changed -> leaf-deleted order.
func TestRoute53ChangeSort(t *testing.T) {
	testTree0 := map[string][]string{
		"fdxn3sn67na5dka4j2gok7bvqi.n": {"enrtree-branch:"},
		"n":                            {"enrtree-root:v1 e=FDXN3SN67NA5DKA4J2GOK7BVQI l=FDXN3SN67NA5DKA4J2GOK7BVQI seq=0 sig=Me7k32DXwG06XmqzNW4b3opMrGXQV9Atv6hmJrVPuNx0QZ-PugeoYlMMndDxyN8cOkWGCxOze26WOyD2JAsAvgE"},
	}

	testTree1 := map[string]string{
		"n":                            "enrtree-root:v1 e=JWXYDBPXYWG6FX3GMDIBFA6CJ4 l=C7HRFPF3BLGF3YR4DY5KX3SMBE seq=1 sig=o908WmNp7LibOfPsr4btQwatZJ5URBr2ZAuxvK4UWHlsB9sUOTJQaGAlLPVAhM__XJesCHxLISo94z5Z2a463gA",
		"C7HRFPF3BLGF3YR4DY5KX3SMBE.n": "enrtree://AM5FCQLWIZX2QFPNJAP7VUERCCRNGRHWZG3YYHIUV7BVDQ5FDPRT2@morenodes.example.org",
		"JWXYDBPXYWG6FX3GMDIBFA6CJ4.n": "enrtree-branch:2XS2367YHAXJFGLZHVAWLQD4ZY,H4FHT4B454P6UXFD7JCYQ5PWDY,MHTDO6TMUBRIA2XWG5LUDACK24",
		"2XS2367YHAXJFGLZHVAWLQD4ZY.n": "enr:-HW4QOFzoVLaFJnNhbgMoDXPnOvcdVuj7pDpqRvh6BRDO68aVi5ZcjB3vzQRZH2IcLBGHzo8uUN3snqmgTiE56CH3AMBgmlkgnY0iXNlY3AyNTZrMaECC2_24YYkYHEgdzxlSNKQEnHhuNAbNlMlWJxrJxbAFvA",
		"H4FHT4B454P6UXFD7JCYQ5PWDY.n": "enr:-HW4QAggRauloj2SDLtIHN1XBkvhFZ1vtf1raYQp9TBW2RD5EEawDzbtSmlXUfnaHcvwOizhVYLtr7e6vw7NAf6mTuoCgmlkgnY0iXNlY3AyNTZrMaECjrXI8TLNXU0f8cthpAMxEshUyQlK-AM0PW2wfrnacNI",
		"MHTDO6TMUBRIA2XWG5LUDACK24.n": "enr:-HW4QLAYqmrwllBEnzWWs7I5Ev2IAs7x_dZlbYdRdMUx5EyKHDXp7AV5CkuPGUPdvbv1_Ms1CPfhcGCvSElSosZmyoqAgmlkgnY0iXNlY3AyNTZrMaECriawHKWdDRk2xeZkrOXBQ0dfMFLHY4eENZwdufn1S1o",
	}

	wantChanges := []*route53.Change{
		{
			Action: sp("CREATE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("2xs2367yhaxjfglzhvawlqd4zy.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enr:-HW4QOFzoVLaFJnNhbgMoDXPnOvcdVuj7pDpqRvh6BRDO68aVi5ZcjB3vzQRZH2IcLBGHzo8uUN3snqmgTiE56CH3AMBgmlkgnY0iXNlY3AyNTZrMaECC2_24YYkYHEgdzxlSNKQEnHhuNAbNlMlWJxrJxbAFvA"`),
				}},
				TTL:  ip(2147483647),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("CREATE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("c7hrfpf3blgf3yr4dy5kx3smbe.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enrtree://AM5FCQLWIZX2QFPNJAP7VUERCCRNGRHWZG3YYHIUV7BVDQ5FDPRT2@morenodes.example.org"`),
				}},
				TTL:  ip(2147483647),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("CREATE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("h4fht4b454p6uxfd7jcyq5pwdy.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enr:-HW4QAggRauloj2SDLtIHN1XBkvhFZ1vtf1raYQp9TBW2RD5EEawDzbtSmlXUfnaHcvwOizhVYLtr7e6vw7NAf6mTuoCgmlkgnY0iXNlY3AyNTZrMaECjrXI8TLNXU0f8cthpAMxEshUyQlK-AM0PW2wfrnacNI"`),
				}},
				TTL:  ip(2147483647),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("CREATE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("jwxydbpxywg6fx3gmdibfa6cj4.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enrtree-branch:2XS2367YHAXJFGLZHVAWLQD4ZY,H4FHT4B454P6UXFD7JCYQ5PWDY,MHTDO6TMUBRIA2XWG5LUDACK24"`),
				}},
				TTL:  ip(2147483647),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("CREATE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("mhtdo6tmubria2xwg5ludack24.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enr:-HW4QLAYqmrwllBEnzWWs7I5Ev2IAs7x_dZlbYdRdMUx5EyKHDXp7AV5CkuPGUPdvbv1_Ms1CPfhcGCvSElSosZmyoqAgmlkgnY0iXNlY3AyNTZrMaECriawHKWdDRk2xeZkrOXBQ0dfMFLHY4eENZwdufn1S1o"`),
				}},
				TTL:  ip(2147483647),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("UPSERT"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp(`"enrtree-root:v1 e=JWXYDBPXYWG6FX3GMDIBFA6CJ4 l=C7HRFPF3BLGF3YR4DY5KX3SMBE seq=1 sig=o908WmNp7LibOfPsr4btQwatZJ5URBr2ZAuxvK4UWHlsB9sUOTJQaGAlLPVAhM__XJesCHxLISo94z5Z2a463gA"`),
				}},
				TTL:  ip(1),
				Type: sp("TXT"),
			},
		},
		{
			Action: sp("DELETE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: sp("fdxn3sn67na5dka4j2gok7bvqi.n"),
				ResourceRecords: []*route53.ResourceRecord{{
					Value: sp("enrtree-branch:"),
				}},
				TTL:  ip(1),
				Type: sp("TXT"),
			},
		},
	}

	var client route53Client
	changes := client.computeChanges("n", testTree1, testTree0)
	if !reflect.DeepEqual(changes, wantChanges) {
		t.Fatalf("wrong changes (got %d, want %d)", len(changes), len(wantChanges))
	}

	wantSplit := [][]*route53.Change{
		wantChanges[:4],
		wantChanges[4:7],
	}
	split := splitChanges(changes, 600)
	if !reflect.DeepEqual(split, wantSplit) {
		t.Fatalf("wrong split batches: got %d, want %d", len(split), len(wantSplit))
	}
}

func sp(s string) *string { return &s }
func ip(i int64) *int64   { return &i }
