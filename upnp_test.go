package upnp

import (
	"testing"
)

func TestIGD(t *testing.T) {
	// connect to router
	d, err := Discover()
	if err != nil {
		t.Fatal(err)
	}

	// discover external IP
	ip, err := d.ExternalIP()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Your external IP is:", ip)

	// forward a port
	err = d.Forward(9001, "upnp test", "TCP")
	if err != nil {
		t.Fatal(err)
	}

	// un-forward a port
	err = d.Clear(9001, "TCP")
	if err != nil {
		t.Fatal(err)
	}

	// record router's location
	loc := d.Location()
	if err != nil {
		t.Fatal(err)
	}

	// connect to router directly
	d, err = Load(loc)
	if err != nil {
		t.Fatal(err)
	}
}
