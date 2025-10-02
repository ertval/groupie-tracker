package data

import "testing"

// TestIntRange_Contains tests the IntRange.Contains method
func TestIntRange_Contains(t *testing.T) {
	tests := []struct {
		name     string
		r        IntRange
		value    int
		expected bool
	}{
		{
			name:     "value within range",
			r:        IntRange{Min: 1, Max: 10},
			value:    5,
			expected: true,
		},
		{
			name:     "value at min boundary",
			r:        IntRange{Min: 1, Max: 10},
			value:    1,
			expected: true,
		},
		{
			name:     "value at max boundary",
			r:        IntRange{Min: 1, Max: 10},
			value:    10,
			expected: true,
		},
		{
			name:     "value below range",
			r:        IntRange{Min: 1, Max: 10},
			value:    0,
			expected: false,
		},
		{
			name:     "value above range",
			r:        IntRange{Min: 1, Max: 10},
			value:    11,
			expected: false,
		},
		{
			name:     "negative range",
			r:        IntRange{Min: -10, Max: -5},
			value:    -7,
			expected: true,
		},
		{
			name:     "zero range (single value)",
			r:        IntRange{Min: 0, Max: 0},
			value:    0,
			expected: true,
		},
		{
			name:     "single value range",
			r:        IntRange{Min: 5, Max: 5},
			value:    5,
			expected: true,
		},
		{
			name:     "single value range miss",
			r:        IntRange{Min: 5, Max: 5},
			value:    6,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Contains(tt.value)
			if got != tt.expected {
				t.Errorf("IntRange{%d, %d}.Contains(%d) = %v, want %v",
					tt.r.Min, tt.r.Max, tt.value, got, tt.expected)
			}
		})
	}
}

// TestIntRange_IsZero tests the IntRange.IsZero method
func TestIntRange_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		r        IntRange
		expected bool
	}{
		{
			name:     "zero range",
			r:        IntRange{Min: 0, Max: 0},
			expected: true,
		},
		{
			name:     "non-zero min only",
			r:        IntRange{Min: 1, Max: 0},
			expected: false,
		},
		{
			name:     "non-zero max only",
			r:        IntRange{Min: 0, Max: 1},
			expected: false,
		},
		{
			name:     "both non-zero",
			r:        IntRange{Min: 1, Max: 10},
			expected: false,
		},
		{
			name:     "negative values",
			r:        IntRange{Min: -5, Max: -1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.IsZero()
			if got != tt.expected {
				t.Errorf("IntRange{%d, %d}.IsZero() = %v, want %v",
					tt.r.Min, tt.r.Max, got, tt.expected)
			}
		})
	}
}

// TestStringSet_Contains tests the StringSet.Contains method
func TestStringSet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		s        StringSet
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			s:        NewStringSet("apple", "banana", "cherry"),
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			s:        NewStringSet("apple", "banana", "cherry"),
			item:     "orange",
			expected: false,
		},
		{
			name:     "empty set",
			s:        NewStringSet(),
			item:     "apple",
			expected: false,
		},
		{
			name:     "empty string in set",
			s:        NewStringSet("", "test"),
			item:     "",
			expected: true,
		},
		{
			name:     "case sensitive",
			s:        NewStringSet("Apple", "Banana"),
			item:     "apple",
			expected: false,
		},
		{
			name:     "whitespace matters",
			s:        NewStringSet("test", "test "),
			item:     "test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Contains(tt.item)
			if got != tt.expected {
				t.Errorf("StringSet.Contains(%q) = %v, want %v",
					tt.item, got, tt.expected)
			}
		})
	}
}

// TestStringSet_IsEmpty tests the StringSet.IsEmpty method
func TestStringSet_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        StringSet
		expected bool
	}{
		{
			name:     "empty set",
			s:        NewStringSet(),
			expected: true,
		},
		{
			name:     "single item",
			s:        NewStringSet("item"),
			expected: false,
		},
		{
			name:     "multiple items",
			s:        NewStringSet("one", "two", "three"),
			expected: false,
		},
		{
			name:     "nil set",
			s:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.IsEmpty()
			if got != tt.expected {
				t.Errorf("StringSet.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIntSet_Contains tests the IntSet.Contains method
func TestIntSet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		s        IntSet
		item     int
		expected bool
	}{
		{
			name:     "item exists",
			s:        NewIntSet(1, 2, 3, 4, 5),
			item:     3,
			expected: true,
		},
		{
			name:     "item does not exist",
			s:        NewIntSet(1, 2, 3, 4, 5),
			item:     10,
			expected: false,
		},
		{
			name:     "empty set",
			s:        NewIntSet(),
			item:     1,
			expected: false,
		},
		{
			name:     "zero in set",
			s:        NewIntSet(0, 1, 2),
			item:     0,
			expected: true,
		},
		{
			name:     "negative numbers",
			s:        NewIntSet(-5, -3, -1, 0, 1),
			item:     -3,
			expected: true,
		},
		{
			name:     "large numbers",
			s:        NewIntSet(1000000, 2000000),
			item:     1000000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Contains(tt.item)
			if got != tt.expected {
				t.Errorf("IntSet.Contains(%d) = %v, want %v",
					tt.item, got, tt.expected)
			}
		})
	}
}

// TestIntSet_IsEmpty tests the IntSet.IsEmpty method
func TestIntSet_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        IntSet
		expected bool
	}{
		{
			name:     "empty set",
			s:        NewIntSet(),
			expected: true,
		},
		{
			name:     "single item",
			s:        NewIntSet(42),
			expected: false,
		},
		{
			name:     "multiple items",
			s:        NewIntSet(1, 2, 3),
			expected: false,
		},
		{
			name:     "nil set",
			s:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.IsEmpty()
			if got != tt.expected {
				t.Errorf("IntSet.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewStringSet tests duplicate handling
func TestNewStringSet(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected int // expected set size
	}{
		{
			name:     "no duplicates",
			items:    []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "with duplicates",
			items:    []string{"a", "b", "a", "c", "b"},
			expected: 3,
		},
		{
			name:     "empty input",
			items:    []string{},
			expected: 0,
		},
		{
			name:     "all duplicates",
			items:    []string{"x", "x", "x"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStringSet(tt.items...)
			if len(got) != tt.expected {
				t.Errorf("NewStringSet(%v) has %d items, want %d",
					tt.items, len(got), tt.expected)
			}
		})
	}
}

// TestNewIntSet tests duplicate handling
func TestNewIntSet(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		expected int // expected set size
	}{
		{
			name:     "no duplicates",
			items:    []int{1, 2, 3},
			expected: 3,
		},
		{
			name:     "with duplicates",
			items:    []int{1, 2, 1, 3, 2},
			expected: 3,
		},
		{
			name:     "empty input",
			items:    []int{},
			expected: 0,
		},
		{
			name:     "all duplicates",
			items:    []int{5, 5, 5},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIntSet(tt.items...)
			if len(got) != tt.expected {
				t.Errorf("NewIntSet(%v) has %d items, want %d",
					tt.items, len(got), tt.expected)
			}
		})
	}
}
