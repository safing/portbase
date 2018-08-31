package bbolt

import bolt "go.etcd.io/bbolt"

db, err := bolt.Open(path, 0666, nil)
if err != nil {
  return err
}
defer db.Close()
