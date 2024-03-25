package beacon

const CreateQueryDBBeacon = `CREATE TABLE IF NOT EXISTS beacon (
	content_id blob PRIMARY KEY,
    content_key blob NOT NULL,
    content_value blob NOT NULL,
    content_size INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS beacon_content_size_idx ON beacon(content_size);`

const InsertQueryBeacon = `INSERT OR IGNORE INTO beacon (content_id, content_key, content_value, content_size)
                            VALUES (?1, ?2, ?3, ?4)`

const DeleteQueryBeacon = `DELETE FROM beacon
    WHERE content_id = (?1)`

const ContentKeyLookupQueryBeacon = `SELECT content_key FROM beacon WHERE content_id = (?1) LIMIT 1`

const ContentValueLookupQueryBeacon = `SELECT content_value FROM beacon WHERE content_id = (?1) LIMIT 1`

const TotalDataSizeQueryBeacon = "SELECT TOTAL(content_size) FROM beacon"

const TotalEntryCountQueryBeacon = "SELECT COUNT(*) FROM beacon"

const ContentSizeLookupQueryBeacon = "SELECT content_size FROM beacon WHERE content_id = (?1)"

const LCUpdateCreateTable = `CREATE TABLE IF NOT EXISTS lc_update (
	period INTEGER PRIMARY KEY,
	value BLOB NOT NULL,
	score INTEGER NOT NULL,
	update_size INTEGER
);
CREATE INDEX IF NOT EXISTS update_size_idx ON lc_update(update_size);
DROP INDEX IF EXISTS period_idx;`

const InsertLCUpdateQuery = `INSERT OR IGNORE INTO lc_update (period, value, score, update_size)
				  VALUES (?1, ?2, ?3, ?4)`

const LCUpdateLookupQuery = `SELECT value FROM lc_update WHERE period = (?1) LIMIT 1`

const LCUpdateLookupQueryByRange = `SELECT value FROM lc_update WHERE period >= (?1) AND period < (?2)`

const LCUpdatePeriodLookupQuery = `SELECT period FROM lc_update WHERE period = (?1) LIMIT 1`

const LCUpdateTotalSizeQuery = `SELECT TOTAL(update_size) FROM lc_update`
