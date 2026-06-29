package spi

import (
	"errors"
	"testing"
)

func ks() []UniqueKey { return []UniqueKey{{ID: "k", Fields: []string{"$.email", "$.age"}}} }

func TestComputeClaims_Full(t *testing.T) {
	c, e := ComputeClaims(ks(), []byte(`{"email":"a@x.com","age":42}`))
	if e != nil || len(c) != 1 {
		t.Fatalf("%+v %v", c, e)
	}
}

func TestComputeClaims_NumCanon(t *testing.T) {
	a, _ := ComputeClaims(ks(), []byte(`{"email":"a","age":42}`))
	b, _ := ComputeClaims(ks(), []byte(`{"email":"a","age":42.0}`))
	d, _ := ComputeClaims(ks(), []byte(`{"email":"a","age":4.2e1}`))
	if a[0].Signature != b[0].Signature || b[0].Signature != d[0].Signature {
		t.Fatalf("42/42.0/4.2e1 must collide: a=%q b=%q d=%q", a[0].Signature, b[0].Signature, d[0].Signature)
	}
}

func TestComputeClaims_BigInt(t *testing.T) {
	a, _ := ComputeClaims(ks(), []byte(`{"email":"a","age":9007199254740993}`))
	b, _ := ComputeClaims(ks(), []byte(`{"email":"a","age":9007199254740992}`))
	if a[0].Signature == b[0].Signature {
		t.Fatal(">2^53 must differ")
	}
}

func TestComputeClaims_TypeTag(t *testing.T) {
	a, _ := ComputeClaims([]UniqueKey{{ID: "k", Fields: []string{"$.v"}}}, []byte(`{"v":"1"}`))
	b, _ := ComputeClaims([]UniqueKey{{ID: "k", Fields: []string{"$.v"}}}, []byte(`{"v":1}`))
	if a[0].Signature == b[0].Signature {
		t.Fatal(`"1" != 1`)
	}
}

func TestComputeClaims_AllNull(t *testing.T) {
	c, e := ComputeClaims(ks(), []byte(`{"email":null,"age":null}`))
	if e != nil || len(c) != 0 {
		t.Fatalf("exempt: %+v %v", c, e)
	}
}

func TestComputeClaims_Partial(t *testing.T) {
	_, e := ComputeClaims(ks(), []byte(`{"email":"a"}`))
	if !errors.Is(e, ErrPartialUniqueKey) {
		t.Fatalf("got %v", e)
	}
}

func TestComputeClaims_OverBound(t *testing.T) {
	_, e := ComputeClaims([]UniqueKey{{ID: "k", Fields: []string{"$.v"}}}, []byte(`{"v":1e1000000000}`))
	if !errors.Is(e, ErrPartialUniqueKey) {
		t.Fatalf("over-bound must reject pre-materialization, got %v", e)
	}
}

func TestComputeClaims_NonScalar(t *testing.T) {
	_, e := ComputeClaims([]UniqueKey{{ID: "k", Fields: []string{"$.v"}}}, []byte(`{"v":{"x":1}}`))
	if !errors.Is(e, ErrPartialUniqueKey) {
		t.Fatalf("non-scalar must reject, got %v", e)
	}
}

func TestComputeClaims_Nested(t *testing.T) {
	c, e := ComputeClaims([]UniqueKey{{ID: "k", Fields: []string{"$.a.b"}}}, []byte(`{"a":{"b":7}}`))
	if e != nil || len(c) != 1 {
		t.Fatalf("nested: %+v %v", c, e)
	}
}
