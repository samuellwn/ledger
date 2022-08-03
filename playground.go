//go:build ignore

/*
Testing playground file.
*/
package main

import "os"

//import "github.com/milochristiansen/ledger"
import "github.com/milochristiansen/ledger/parse"

func main() {
	transactions, err := parse.ParseLedger(parse.NewCharReader(`
2021/09/25 * Youtube Premium
    Expenses:Entertainment:Youtube Premium     $19.15
    Liabilities:CreditCard:Discover

2021/09/26 * 6 month avi sub
    Expenses:Entertainment:Twitch              $32.12
    Liabilities:CreditCard:Discover

2021/09/27 * CMS
    Expenses:Parental:Internet                 $49.95
    Liabilities:CreditCard:Discover

2021/09/29 * Gas
    Expenses:Car:Gas                           $16.00
    Liabilities:CreditCard:Discover

2021/09/29 * Amazon Prime
    Expenses:Entertainment:Amazon Prime        $13.83
    Liabilities:CreditCard:Discover

2021/10/01 * W4sted Patreon
    Expenses:Entertainment:Patreon             $5.00
    Liabilities:CreditCard:Discover

2021/10/01 * FU Everence.
    Expenses:Other                             $5.00
    Assets:Everence:Checking

2021/10/03 * Walmart & Pretzel
    Expenses:Food                              $90.81
    Assets:Wallet                             $-12.00
    Liabilities:CreditCard:Discover

2021/10/04 * Forgotten Weapons
    Expenses:Entertainment:Patreon             $5.00
    Liabilities:CreditCard:Discover

2021/10/04 * Timberborn
    Expenses:Entertainment:Games               $26.61
    Liabilities:CreditCard:Discover

2021/10/09 * Work Email
    Expenses:Work Email                        $6.38
    Liabilities:CreditCard:Discover
`, 15))
	if err != nil {
		panic(err)
	}

	transactions.Format(os.Stdout)
}
