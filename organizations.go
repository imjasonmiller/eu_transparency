package main

import (
	"database/sql"
	"encoding/xml"
	"os"

	"github.com/lib/pq"
)

type organization struct {
	IdentificationCode string `xml:"identificationCode"`
	Name               struct {
		OriginalName string `xml:"originalName"`
	} `xml:"name"`
	ContactDetails struct {
		Country     string `xml:"country"`
		CountryCode int
	} `xml:"contactDetails"`
	LegalStatus      string `xml:"legalStatus"`
	RegistrationDate string `xml:"registrationDate"`
	LastUpdateDate   string `xml:"lastUpdateDate"`
}

func processXML(file string, db *sql.DB) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	countries, err := countryNameToID(db)
	if err != nil {
		return err
	}

	orgs := &[]organization{}
	dec := xml.NewDecoder(f)

	// Counter tracks interestRepresentatives.
	var counter int64

	for {
		// Stream and read tokens from the .xml file.
		t, _ := dec.Token()
		if t == nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "interestRepresentative" {
				var org organization

				dec.DecodeElement(&org, &se)

				if code, ok := countries[org.ContactDetails.Country]; ok {
					org.ContactDetails.CountryCode = code
				}

				// Bulk upsert every 1000 rows.
				if counter%1000 == 0 && counter != 0 {
					err := bulkUpsertOrganizations(orgs, db)
					if err != nil {
						return err
					}
					orgs = nil
				}

				*orgs = append(*orgs, org)
				counter++
			}
		}
	}

	// Upsert the remainder
	if len(*orgs) != 0 {
		err := bulkUpsertOrganizations(orgs, db)
		if err != nil {
			return err
		}
	}

	return nil
}

func bulkUpsertOrganizations(orgs *[]organization, db *sql.DB) error {
	// Start transaction.
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	// Allow for a rollback if the transaction was not succesfull.
	success := false

	defer func() {
		if !success {
			txn.Rollback()
		}
	}()

	// Create a temporary table that is dropped on commit.
	// It allows for a INSERT INTO ... ON CONFLICT
	_, err = txn.Exec(`CREATE TEMP TABLE organizations_temp (
		organization_id             TEXT NOT NULL,
		organization_name           TEXT NOT NULL,
		organization_country		INT  NOT NULL,
		organization_legal_status   TEXT NOT NULL,
		organization_updated_at     TIMESTAMP WITH TIME ZONE NOT NULL,
		organization_registered_at  TIMESTAMP WITH TIME ZONE NOT NULL
	)	ON COMMIT DROP`)
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("organizations_temp",
		"organization_id",
		"organization_name",
		"organization_country",
		"organization_legal_status",
		"organization_updated_at",
		"organization_registered_at",
	))
	if err != nil {
		return err
	}

	// Copy into temporary table
	for _, org := range *orgs {
		_, err := stmt.Exec(
			org.IdentificationCode,
			org.Name.OriginalName,
			org.ContactDetails.CountryCode,
			org.LegalStatus,
			org.LastUpdateDate,
			org.RegistrationDate,
		)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	// Insert from temp to real table
	_, err = txn.Exec(`
		INSERT INTO organizations
		SELECT * FROM organizations_temp
		ON CONFLICT (organization_id)
		DO UPDATE SET
			organization_id	= EXCLUDED.organization_id,
			organization_name = EXCLUDED.organization_name,
			organization_legal_status = EXCLUDED.organization_legal_status,
			organization_country = EXCLUDED.organization_country,
			organization_updated_at = EXCLUDED.organization_updated_at,
			organization_registered_at = EXCLUDED.organization_registered_at
		WHERE EXCLUDED.organization_updated_at > organizations.organization_updated_at
	`)
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	success = true

	return nil
}

// Returns a map of country names to the corresponding ID in the database.
func countryNameToID(db *sql.DB) (map[string]int, error) {
	countries := map[string]int{}

	rows, err := db.Query(`
		SELECT country_names.country_name, countries.country_id
		FROM countries
		INNER JOIN country_names
		ON countries.country_code = country_names.country_code;
	`)
	if err != nil {
		return countries, err
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		var id int

		if err := rows.Scan(&name, &id); err != nil {
			return countries, err
		}

		countries[name] = id
	}
	if err := rows.Err(); err != nil {
		return countries, err
	}

	return countries, nil
}
