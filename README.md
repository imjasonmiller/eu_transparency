### 🇪🇺 European Union's meetings

An index of all meetings between organizations and commissioners and their cabinet members.

---

The European Union's [Transparency Register](http://ec.europa.eu/transparencyregister/public/homePage.do) aims to make the decision-making process as transparent as possible. One of the requirements for commissioners and their cabinet members is that they can only have meetings with organizations that are included in the transparency register and that these meetings are to be written down.

As the meetings for each commissioner or their cabinet members were only available in a paginated HTML format, I attempted to present all of the data in a more meaningful way.


### Install

Create the `eu_transparency` database by running:

    psql -U [username] -h [host] -p [port] --file=database/schema.sql

### Build 

```
go build
```

