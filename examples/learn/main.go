package main

import "fmt"

type PaymentMethod interface {
	Pay(amount int) string
}

type Cash struct{}

func (Cash) Pay(amount int) string {
	return fmt.Sprintf("Paid %d in cash", amount)
}

type Card struct {
	Bank string
}

func (c Card) Pay(amount int) string {
	return fmt.Sprintf("Paid %d with %s card", amount, c.Bank)
}

type UPI struct {
	ID string
}

func (u UPI) Pay(amount int) string {
	return fmt.Sprintf("Paid %d via UPI (%s)", amount, u.ID)
}

func checkout(method PaymentMethod, amount int) {
	fmt.Println(method.Pay(amount))
}

func main() {
	checkout(Cash{}, 100)
	checkout(Card{Bank: "HDFC"}, 250)
	checkout(UPI{ID: "arjun@upi"}, 75)

	fmt.Println("---")

	wallet := []PaymentMethod{
		Cash{},
		Card{Bank: "SBI"},
		UPI{ID: "gpay@ok"},
	}
	for _, method := range wallet {
		checkout(method, 500)
	}
}
