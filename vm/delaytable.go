package vm

import "github.com/vsariola/sointu"

// findSuperIntArray finds a small super array containing all
// the subarrays passed to it. Returns the super array and indices where
// the subarrays can be found. For example:
//   FindSuperIntArray([][]int{{4,5,6},{1,2,3},{3,4}})
// returns {1,2,3,4,5,6},{3,0,2}
// Implemented using a greedy search, so does not necessarily find
// the true optimal (the problem is NP-hard and analogous to traveling
// salesman problem).
//
// Used to construct a small delay time table without unnecessary repetition
// of delay times.
func findSuperIntArray(arrays [][]int) ([]int, []int) {
	// If we go past MAX_MERGES, the algorithm could get slow and hang the computer
	// So this is a safety limit: after this problem size, just merge any arrays
	// until we get into more manageable range
	const maxMerges = 1000
	min := func(a int, b int) int {
		if a < b {
			return a
		}
		return b
	}
	overlap := func(a []int, b []int) (int, int) {
		minShift := len(a)
		for shift := len(a) - 1; shift >= 0; shift-- {
			overlapping := true
			for k := shift; k < min(len(a), len(b)+shift); k++ {
				if a[k] != b[k-shift] {
					overlapping = false
					break
				}
			}
			if overlapping {
				minShift = shift
			}
		}
		overlap := min(len(a)-minShift, len(b))
		return overlap, minShift
	}
	sliceNumbers := make([]int, len(arrays))
	startIndices := make([]int, len(arrays))
	var processedArrays [][]int
	for i := range arrays {
		if len(arrays[i]) == 0 {
			// Zero length arrays do not need to be processed at all
			// They will 'start' at index 0 always as they have no length.
			sliceNumbers[i] = -1
		} else {
			sliceNumbers[i] = len(processedArrays)
			processedArrays = append(processedArrays, arrays[i])
		}
	}
	if len(processedArrays) == 0 {
		return []int{}, startIndices // no arrays with len>0 to process, just return empty array and all indices as 0
	}
	for len(processedArrays) > 1 { // there's at least two candidates that could be be merged
		maxO, maxI, maxJ, maxS := -1, -1, -1, -1
		if len(processedArrays) < maxMerges {
			// find the pair i,j that results in the largest overlap with array i coming first, followed by potentially overlapping array j
			for i := range processedArrays {
				for j := range processedArrays {
					if i == j {
						continue
					}
					overlap, shift := overlap(processedArrays[i], processedArrays[j])
					if overlap > maxO {
						maxI, maxJ, maxO, maxS = i, j, overlap, shift
					}
				}
			}
		} else {
			// The task is daunting, we have over MAX_MERGES overlaps to test. Just merge two first ones until the task is more manageable size
			overlap, shift := overlap(processedArrays[0], processedArrays[1])
			maxI, maxJ, maxO, maxS = 0, 1, overlap, shift
		}
		for k := range sliceNumbers {
			if sliceNumbers[k] == maxJ {
				// update slice pointers to point maxI instead of maxJ (maxJ will  be appended to maxI, taking overlap into account)
				sliceNumbers[k] = maxI
				startIndices[k] += maxS // the array j starts at index maxS in array i
			}
			if sliceNumbers[k] > maxJ {
				// pointers maxJ reduced by 1 as maxJ will be deleted
				sliceNumbers[k]--
			}
		}
		// if array j was not entirely included within array j
		if maxO < len(processedArrays[maxJ]) {
			// append array maxJ to array maxI, without duplicating the overlapping part
			processedArrays[maxI] = append(processedArrays[maxI], processedArrays[maxJ][maxO:]...)
		}
		// finally, remove element maxJ from processedArrays
		processedArrays = append(processedArrays[:maxJ], processedArrays[maxJ+1:]...)
	}
	return processedArrays[0], startIndices // there should be only one slice left in the arrays after the loop
}

// constructDelayTimeTable tries to construct the delay times table abusing
// overlapping between different delay times tables as much as possible.
// Especially: if two delay units use exactly the same delay times, they appear
// in the table only once.
//
// Returns the delay time table and two dimensional array of integers where
// element [i][u] is the index for instrument i / unit u in the delay table if
// the unit was a delay unit. For non-delay untis, the element is just 0.
func constructDelayTimeTable(patch sointu.Patch, bpm int) ([]int, [][]int) {
	ind := make([][]int, len(patch))
	var subarrays [][]int
	// flatten the delay times into one array of arrays
	// saving the indices where they were placed
	for i, instr := range patch {
		ind[i] = make([]int, len(instr.Units))
		for j, unit := range instr.Units {
			// only include delay times for delays. Only delays should use delay
			// times. Only delay times for enabled delay units should be in the
			// table.
			if unit.Type == "delay" && !unit.Disabled {
				ind[i][j] = len(subarrays)
				converted := make([]int, len(unit.VarArgs))
				copy(converted, unit.VarArgs)
				if unit.Parameters["notetracking"] == 2 {
					for i, t := range converted {
						delay := 44100 * 60 * t / 48 / bpm
						if delay > 65535 {
							delay = 65535
						}
						converted[i] = delay
					}
				}
				subarrays = append(subarrays, converted)
			}
		}
	}
	delayTable, indices := findSuperIntArray(subarrays)
	// cancel the flattening, so unitindices can be used to
	// to find the index of each delay in the delay table
	unitindices := make([][]int, len(patch))
	for i, instr := range patch {
		unitindices[i] = make([]int, len(instr.Units))
		for j, unit := range instr.Units {
			if unit.Type == "delay" && !unit.Disabled {
				unitindices[i][j] = indices[ind[i][j]]
			}
		}
	}
	return delayTable, unitindices
}
