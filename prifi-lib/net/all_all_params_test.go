package net

import "testing"

func TestUtils(t *testing.T) {

	b := NewALL_ALL_PARAMETERS_BUILDER()

	b.Add("key1", "val1")
	b.Add("key2", 123)
	b.Add("key3", true)

	if b.data["key1"] != "val1" {
		t.Error("key1 should equals val1")
	}
	if b.data["key2"] != 123 {
		t.Error("key2 should equals 123")
	}
	if b.data["key3"] != true {
		t.Error("key3 should equals true")
	}

	m := b.BuildMessage(true)

	if m.ForceParams != true {
		t.Error("ForceParams should be true")
	}
	if m.Params["key1"].Data != "val1" {
		t.Error("key1 should equals val1")
	}
	if m.Params["key2"].Data != 123 {
		t.Error("key2 should equals 123")
	}
	if m.Params["key3"].Data != true {
		t.Error("key3 should equals true")
	}


	if m.StringValueOrElse("key1", "else") != "val1" {
		t.Error("key1 should equals val1")
	}
	if m.IntValueOrElse("key2", 324) != 123 {
		t.Error("key2 should equals 123")
	}
	if m.BoolValueOrElse("key3", false) != true {
		t.Error("key3 should equals true")
	}
	if m.ValueOrElse("key1", Interface{}) != "val1" {
		t.Error("key1 should equals val1")
	}
	emptyInterface := Interface{}
	if m.ValueOrElse("key5", Interface{}) != emptyInterface {
		t.Error("should return the empty interface")
	}
	if m.StringValueOrElse("key5", "else") != "else" {
		t.Error("non-existent key should return elseVal")
	}
	if m.IntValueOrElse("key6", 324) != 324 {
		t.Error("non-existent key should return elseVal")
	}
	if m.BoolValueOrElse("key7", false) != false {
		t.Error("non-existent key should return elseVal")
	}
}
