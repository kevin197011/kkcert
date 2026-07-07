package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

// DeleteCertificatesForDomain physically removes all certs for a domain except keepID (empty = delete all).
func (s *Store) DeleteCertificatesForDomain(domainID, keepID string) (int, error) {
	var n int
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketCertificates)
		var keys [][]byte
		_ = b.ForEach(func(k, v []byte) error {
			var c Certificate
			if json.Unmarshal(v, &c) != nil {
				return nil
			}
			if c.DomainID == domainID && (keepID == "" || c.ID != keepID) {
				keys = append(keys, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range keys {
			if err := b.Delete(k); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}

// CleanupStaleData removes orphaned and invalid records. Returns counts removed.
func (s *Store) CleanupStaleData() (certs, sessions int, err error) {
	archived := map[string]bool{}
	domains, _ := s.ListDomains(true)
	for _, d := range domains {
		if d.Archived {
			archived[d.ID] = true
		}
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		cb := tx.Bucket(bucketCertificates)
		var certKeys [][]byte
		_ = cb.ForEach(func(k, v []byte) error {
			var c Certificate
			if json.Unmarshal(v, &c) != nil {
				certKeys = append(certKeys, append([]byte{}, k...))
				return nil
			}
			if archived[c.DomainID] || !c.Active {
				certKeys = append(certKeys, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range certKeys {
			if err := cb.Delete(k); err != nil {
				return err
			}
			certs++
		}

		sb := tx.Bucket(bucketSessions)
		now := time.Now()
		var sessKeys [][]byte
		_ = sb.ForEach(func(k, v []byte) error {
			var sess Session
			if json.Unmarshal(v, &sess) != nil {
				sessKeys = append(sessKeys, append([]byte{}, k...))
				return nil
			}
			if now.After(sess.ExpiresAt) {
				sessKeys = append(sessKeys, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range sessKeys {
			if err := sb.Delete(k); err != nil {
				return err
			}
			sessions++
		}
		return nil
	})
	return certs, sessions, err
}
