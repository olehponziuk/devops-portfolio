package main

import "testing"

func TestMain(t *testing.T) {
    // Simple test that always passes
    if 1+1 != 2 {
        t.Error("Math is broken!")
    }
}
