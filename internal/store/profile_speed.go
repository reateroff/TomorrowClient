package store

import (
	"TomorrowClient/internal/model"
	"reflect"
)

func sameSpeedProfile(a, b model.Profile) bool {
	a.Speed = nil
	b.Speed = nil
	return reflect.DeepEqual(a, b)
}

// SaveProfileSpeed refuses stale measurements if a profile or its subscription changed mid-test.
func (s *Store) SaveProfileSpeed(expected model.Profile, subUpdatedAt int64, result model.ProfileSpeed) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expected.SubID != "" {
		found := false
		for _, sub := range s.subs {
			if sub.ID == expected.SubID {
				found = sub.UpdatedAt == subUpdatedAt
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	for i, p := range s.profiles {
		if p.ID == expected.ID {
			if !sameSpeedProfile(p, expected) {
				return false, nil
			}
			previous := p.Speed
			s.profiles[i].Speed = &result
			if err := writeJSON(s.profilesPath(), s.profiles); err != nil {
				s.profiles[i].Speed = previous
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}
