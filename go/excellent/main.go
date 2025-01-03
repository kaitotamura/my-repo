package main

// EvenOrOdd checks if a number is even or odd.
func EvenOrOdd(number int) string {
	// If the number is divisible by 2, it's even.
	if number%2 == 0 {
		return "even"
	} else {
		// Otherwise, it's odd.
		return "odd"
	}
}
