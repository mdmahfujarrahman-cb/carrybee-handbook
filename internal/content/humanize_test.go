package content

import "testing"

func TestHumanStatus(t *testing.T) {
	cases := []struct{ in, want string }{
		{"TransferStatusAtSubSort", "At sub sort"},
		{"ReceivedAtLastMileHub", "Received at last mile hub"},
		{"TransferStatusDelivered", "Delivered"},
		{"Delivered", "Delivered"},
		{"OnHold", "On hold"},
		{"TransferStatus", "Transfer status"},
		{"", ""},
	}
	for _, c := range cases {
		if got := HumanStatus(c.in); got != c.want {
			t.Errorf("HumanStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanText(t *testing.T) {
	cases := map[string]string{"on_the_way_to_lmh": "On the way to lmh", "delivered": "Delivered", "": ""}
	for in, want := range cases {
		if got := HumanText(in); got != want {
			t.Errorf("HumanText(%q) = %q, want %q", in, got, want)
		}
	}
}
