package httpapi

import "testing"

func TestOrderTransitions(t *testing.T) {
	valid := [][2]string{{"pending_payment", "esim_assigned"}, {"esim_assigned", "activated"}, {"activated", "completed"}}
	for _, transition := range valid {
		if !allowedOrderTransitions[transition[0]][transition[1]] {
			t.Fatalf("expected valid transition %s -> %s", transition[0], transition[1])
		}
	}
	if allowedOrderTransitions["completed"]["cancelled"] {
		t.Fatal("completed order must not be cancellable")
	}
}
