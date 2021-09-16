package shardctrler

func getMinKeyOfMinValue(cnt map[int]int) int {
	choose := -1
	minCnt := 99999999
	for k, v := range(cnt) {
		if v < minCnt {
			choose = k
			minCnt = v
		} else if v == minCnt && k < choose {
			choose = k
		}
	}
	return choose
}

func getMinKeyOfMaxValue(cnt map[int]int) int {
	choose := -1
	minCnt := -99999999
	for k, v := range(cnt) {
		if v > minCnt {
			choose = k
			minCnt = v
		} else if v == minCnt && k < choose {
			choose = k
		}
	}
	return choose
}

func findFirstIndex(shards [NShards]int, x int) int {
	for i := 0; i < NShards; i++ {
		if shards[i] == x {
			return i
		}
	}
	return -1
}

func reblance(shards [NShards]int, groups map[int][]string) [NShards]int {
	var cnt map[int]int = make(map[int]int)
	for k, _ := range groups {
		cnt[k] = 0
	}
	for _, gid := range shards {
		_, ok := groups[gid]
		if ok {
			cnt[gid]++
		}
	}
	
	// check if any shard run on a removed gid
	for shard := 0; shard < NShards; shard++ {
		gid := shards[shard]
		_, ok := groups[gid]
		if !ok {
			// find a gid with minimum cnt and assign this shard to it
			choose := getMinKeyOfMinValue(cnt)
			shards[shard] = choose
			cnt[choose]++
		}
	}

	for {
		maxGid := getMinKeyOfMaxValue(cnt)
		minGid := getMinKeyOfMinValue(cnt)
		if cnt[maxGid] - cnt[minGid] > 1 {
			i := findFirstIndex(shards, maxGid)
			shards[i] = minGid
			cnt[maxGid]--
			cnt[minGid]++
		} else {
			break
		}
	}
	return shards
}