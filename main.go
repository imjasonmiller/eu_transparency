package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := loadConfig(".env")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := databaseConn(cfg)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("test"))

	fmt.Println("id:", id)

	// if err := conn.Backup(); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := upsertDepartments(conn.db); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := meetings(conn.db); err != nil {
	// 	log.Fatal(err)
	// }

	// src := "http://ec.europa.eu/transparencyregister/public/consultation/statistics.do?action=getLobbyistsXml&fileType=NEW"
	// dst := "test.xml"

	// if err := downloadFile(src, dst); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := processXML(dst, conn.db); err != nil {
	// 	log.Fatal(err)
	// }

}
