package cmd

func progressPercent(done int, total int) int {
	if total <= 0 {
		return 100
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	return done * 100 / total
}
