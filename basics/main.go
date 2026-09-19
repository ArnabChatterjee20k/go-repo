package main

import "fmt"

func change(s []int) {
	s = append(s, 100)
	s[0] = 12
}

func change_pointer(s *[]int) {
	*s = append(*s, 100)
}

func sum(nums ...int) int {
	total := 0
	for _, v := range nums {
		total += v
	}
	return total
}

// closure
func counter() func() int {
	i := 0
	return func() int {
		i++
		return i
	}
}

func main() {
	s := []int{1, 23, 4, 5}
	change(s)
	fmt.Println("slice: ", s)

	s_with_cap := make([]int, 4, 10)
	change(s_with_cap)
	fmt.Println("slice: ", s_with_cap)

	fmt.Println(sum(1, 2, 3, 4, 5))

	c := counter()
	c()
	c()
	c()
	fmt.Println(c())

	// passing tight cap so that append might creates a new due to the cap is filled
	change_pointer(&s)
	fmt.Println("slice: ", s)
}
