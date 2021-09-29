package store

import (
	"bytes"
	"fmt"
	"math"
)

type TagPopulator struct {
	DBConnection Database
}

func (t *TagPopulator) PopulateTables(tl int) error {
	var err error

	row := t.DBConnection.QueryRow(`SELECT COUNT(*) FROM "groups"`)
	if row != nil {
		var count int
		err = row.Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}

	var b bytes.Buffer
	_, err = b.WriteString(`INSERT INTO "groups" (guid) VALUES (NULL)`)
	if err != nil {
		return err
	}

	for i := 1; i < int(math.Exp2(float64(tl*8)))-1; i++ {
		_, err = b.WriteString(", (NULL)")
		if err != nil {
			return err
		}
	}

	_, err = t.DBConnection.Exec(b.String())
	if err != nil {
		return fmt.Errorf("populating tables: %s", err)
	}

	return nil
}
