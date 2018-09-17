package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

type department struct {
	Name         string   `json:"name"`
	Abbreviation string   `json:"abbreviation"`
	Description  string   `json:"description"`
	Leaders      []leader `json:"leaders"`
	Members      []member `json:"members"`
}

type leader struct {
	ID           *string `json:"id"`
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	Country      string  `json:"country"`
	LeaderHostID string  `json:"leaderHostId"`
	MemberHostID string  `json:"memberHostId"`
}

type member struct {
	ID    *string `json:"id"`
	Name  string  `json:"name"`
	Roles []struct {
		Leader *string `json:"leader"`
		Role   string  `json:"role"`
	} `json:"roles"`
}

// Returns a map of ISO 3166-1 alpha 2 codes to the corresponding ID in the database.
func countryCodeToID(db *sql.DB) (map[string]int, error) {
	countries := map[string]int{}

	rows, err := db.Query(`SELECT country_code, country_id FROM countries`)
	if err != nil {
		return countries, err
	}

	defer rows.Close()

	for rows.Next() {
		var code string
		var id int

		if err := rows.Scan(&code, &id); err != nil {
			return countries, err
		}

		countries[code] = id
	}
	if err := rows.Err(); err != nil {
		return countries, err
	}

	return countries, nil
}

func forEachDepartment(dir string, fn func(dep department) error) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		var dep department

		// Read file data.
		data, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}

		// Unmarshal file data into dep variable.
		if err := json.Unmarshal(data, &dep); err != nil {
			return err
		}

		// Apply function to each department.
		fn(dep)
	}
	return nil
}

// Upsert the contents of each department into the database.
func upsertDepartments(db *sql.DB) error {
	countries, err := countryCodeToID(db)
	if err != nil {
		return err
	}

	err = forEachDepartment(filepath.Join("database", "departments"), func(dep department) error {
		// Upsert departments.
		_, err := db.Exec(`
            INSERT INTO departments (department_abbreviation, department_name, department_description)
            VALUES ($1, $2, $3)
            ON CONFLICT (department_abbreviation) DO UPDATE SET
                department_name = EXCLUDED.department_name,
                department_description = EXCLUDED.department_description`,
			dep.Abbreviation, dep.Name, dep.Description,
		)
		if err != nil {
			return err
		}

		// Upsert all leaders.
		for _, leader := range dep.Leaders {
			var country int

			if id, ok := countries[leader.Country]; ok {
				country = id
			}

			_, err := db.Exec(`
                INSERT INTO leaders (leader_id, leader_name, leader_role, leader_country, leader_department)
                VALUES ($1, $2, $3, $4, $5)
                ON CONFLICT (leader_id) DO UPDATE SET
                    leader_name = EXCLUDED.leader_name,
                    leader_role = EXCLUDED.leader_role,
                    leader_country = EXCLUDED.leader_country,
                    leader_department = EXCLUDED.leader_department`,
				*leader.ID, leader.Name, leader.Role, country, dep.Abbreviation,
			)
			if err != nil {
				return err
			}
		}

		// Upsert all members.
		for _, member := range dep.Members {
			_, err := db.Exec(`
                INSERT INTO members (member_id, member_name)
                VALUES ($1, $2)
                ON CONFLICT (member_id) DO UPDATE SET
                    member_name = EXCLUDED.member_name`,
				*member.ID, member.Name,
			)
			if err != nil {
				return err
			}

			for _, role := range member.Roles {
				_, err := db.Exec(`
                    INSERT INTO members_roles(leader_id, member_id, member_role)
                    VALUES ($1, $2, $3)
                    ON CONFLICT DO NOTHING`,
					*role.Leader, *member.ID, role.Role,
				)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
