package lookup

import "context"

// FluzCapacitorAlgorithm works by narrowing the epoch search area if an update is found
// going back and forth in time
// First, it will attempt to find an update where it should be now if the hint was
// really the last update. If that lookup fails, then the last update must be either the hint itself
// or the epochs right below. If however, that lookup succeeds, then the update must be
// that one or within the epochs right below.
// see the guide for a more graphical representation
func FluzCapacitorAlgorithm(ctx context.Context, now uint64, hint Epoch, read ReadFunc) (value interface{}, err error) {
	var lastFound interface{}
	var epoch Epoch
	if hint == NoClue {
		hint = worstHint
	}

	t := now

	for {
		epoch = GetNextEpoch(hint, t)
		value, err = read(ctx, epoch, now)
		if err != nil {
			return nil, err
		}
		if value != nil {
			lastFound = value
			if epoch.Level == LowestLevel || epoch.Equals(hint) {
				return value, nil
			}
			hint = epoch
			continue
		}
		if epoch.Base() == hint.Base() {
			if lastFound != nil {
				return lastFound, nil
			}
			// we have reached the hint itself
			if hint == worstHint {
				return nil, nil
			}
			// check it out
			value, err = read(ctx, hint, now)
			if err != nil {
				return nil, err
			}
			if value != nil {
				return value, nil
			}
			// bad hint.
			t = hint.Base()
			hint = worstHint
			continue
		}
		base := epoch.Base()
		if base == 0 {
			return nil, nil
		}
		t = base - 1
	}

}
