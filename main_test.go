package main

import "testing"

func TestMain(t *testing.T) {
    if 1+1 != 2 {
        t.Error("Math is broken!")
    }
}

func TestPassing(t *testing.T) {
    if 2+2 == 4 {
        t.Log("Math is working correctly!")
    }
}
