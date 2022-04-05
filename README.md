
# Ledger

Package Ledger contains a parser for Ledger CLI transactions.

This should support the spec more-or-less fully for simple transactions,
but I did not add support for automated transactions or budgeting.

Additionally, I properly implemented String on everything so you can dump
Transactions to a file and read it with Ledger again.

Finally, there are a bunch of functions and methods for dealing with
transactions that should be helpful to anyone trying to use this for
real work.
