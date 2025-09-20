package vtec

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twpayne/go-geos"
)

// Find an event based on VTEC parameters
func FindEvent(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int) (*VTECEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Lets check if the VTEC Event is already in the database
	rows, err := db.Query(ctx, `
			SELECT * FROM vtec.events WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan the row into the event struct
	event := &VTECEvent{}
	if rows.Next() {
		if err := event.scan(rows); err != nil {
			return nil, err
		}
	} else {
		event = nil
	}

	return event, nil
}

// Database model for a VTEC event
type VTECEvent struct {
	ID           int        `json:"id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends"`
	EndInitial   time.Time  `json:"end_initial"`
	Class        string     `json:"class"`
	Phenomena    string     `json:"phenomena"`
	WFO          string     `json:"wfo"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	Year         int        `json:"year"`
	Title        string     `json:"title"`
	IsEmergency  bool       `json:"is_emergency"`
	IsPDS        bool       `json:"is_pds"`
}

// Scan a row into the event struct
func (event *VTECEvent) scan(rows pgx.Rows) error {
	return rows.Scan(
		&event.ID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.Issued,
		&event.Starts,
		&event.Expires,
		&event.Ends,
		&event.EndInitial,
		&event.Class,
		&event.Phenomena,
		&event.WFO,
		&event.Significance,
		&event.EventNumber,
		&event.Year,
		&event.Title,
		&event.IsEmergency,
		&event.IsPDS,
	)
}

// Insert the VTEC event into the database
func (event *VTECEvent) Insert(db *pgxpool.Pool) error {

	rows, err := db.Query(context.Background(), `
				INSERT INTO vtec.events(issued, starts, expires, ends, end_initial, class, phenomena, wfo, 
				significance, event_number, year, title, is_emergency, is_pds) VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
				`, event.Issued, event.Starts, event.Expires, event.Ends, event.EndInitial, event.Class,
		event.Phenomena, event.WFO, event.Significance, event.EventNumber, event.Year, event.Title,
		event.IsEmergency, event.IsPDS)

	if err != nil {
		return err
	}

	defer rows.Close()

	// Scan the row into the event struct
	if rows.Next() {
		if err := event.scan(rows); err != nil {
			return err
		}
	}

	return nil
}

// Database model for a VTEC update
type VTECUpdate struct {
	ID            int        `json:"id"`
	CreatedAt     time.Time  `json:"created_at,omitempty"`
	Issued        time.Time  `json:"issued"`
	Starts        *time.Time `json:"starts,omitempty"`
	Expires       time.Time  `json:"expires"`
	Ends          time.Time  `json:"ends,omitempty"`
	Text          string     `json:"text"`
	Product       string     `json:"product"`
	WFO           string     `json:"wfo"`
	Action        string     `json:"action"`
	Class         string     `json:"class"`
	Phenomena     string     `json:"phenomena"`
	Significance  string     `json:"significance"`
	EventNumber   int        `json:"event_number"`
	Year          int        `json:"year"`
	Title         string     `json:"title"`
	IsEmergency   bool       `json:"is_emergency"`
	IsPDS         bool       `json:"is_pds"`
	Geom          *geos.Geom `json:"geom,omitempty"`
	Direction     *int       `json:"direction"`
	Location      *geos.Geom `json:"location"`
	Speed         *int       `json:"speed"`
	SpeedText     *string    `json:"speed_text"`
	TMLTime       *time.Time `json:"tml_time"`
	UGC           []string   `json:"ugc"`
	Tornado       string     `json:"tornado,omitempty"`
	Damage        string     `json:"damage,omitempty"`
	HailThreat    string     `json:"hail_threat,omitempty"`
	HailTag       string     `json:"hail_tag,omitempty"`
	WindThreat    string     `json:"wind_threat,omitempty"`
	WindTag       string     `json:"wind_tag,omitempty"`
	FlashFlood    string     `json:"flash_flood,omitempty"`
	RainfallTag   string     `json:"rainfall_tag,omitempty"`
	FloodTagDam   string     `json:"flood_tag_dam,omitempty"`
	SpoutTag      string     `json:"spout_tag,omitempty"`
	SnowSquall    string     `json:"snow_squall,omitempty"`
	SnowSquallTag string     `json:"snow_squall_tag,omitempty"`
}

func (update *VTECUpdate) scan(row pgx.Row) error {
	return row.Scan(
		&update.ID,
		&update.CreatedAt,
		&update.Issued,
		&update.Starts,
		&update.Expires,
		&update.Ends,
		&update.Text,
		&update.Product,
		&update.WFO,
		&update.Action,
		&update.Class,
		&update.Phenomena,
		&update.Significance,
		&update.EventNumber,
		&update.Year,
		&update.Title,
		&update.IsEmergency,
		&update.IsPDS,
		&update.Geom,
		&update.Direction,
		&update.Location,
		&update.Speed,
		&update.SpeedText,
		&update.TMLTime,
		&update.UGC,
		&update.Tornado,
		&update.Damage,
		&update.HailThreat,
		&update.HailTag,
		&update.WindThreat,
		&update.WindTag,
		&update.FlashFlood,
		&update.RainfallTag,
		&update.FloodTagDam,
		&update.SpoutTag,
		&update.SnowSquall,
		&update.SnowSquallTag,
	)
}

func (update *VTECUpdate) Insert(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	INSERT INTO vtec.updates(issued, starts, expires, ends, text, product, 
	wfo, action, class, phenomena, significance, event_number, year, title, 
	is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
	ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
	rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 
	$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
	`, update.Issued, update.Starts, update.Expires, update.Ends, update.Text, update.Product,
		update.WFO, update.Action, update.Class, update.Phenomena, update.Significance, update.EventNumber, update.Year, update.Title,
		update.IsEmergency, update.IsPDS, update.Geom, update.Direction, update.Location, update.Speed, update.SpeedText, update.TMLTime,
		update.UGC, update.Tornado, update.Damage, update.HailThreat, update.HailTag, update.WindThreat, update.WindTag, update.FlashFlood,
		update.RainfallTag, update.FloodTagDam, update.SpoutTag, update.SnowSquall, update.SnowSquallTag)
	if err != nil {
		return err
	}

	if rows.Next() {
		if err := update.scan(rows); err != nil {
			return err
		}
	}

	return nil
}

func FindCurrentUGCsForEvent(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int, expires time.Time) ([]*VTECUGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM vtec.ugcs WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5 AND action NOT IN ('CAN', 'UPG') AND expires > $6
	`, wfo, phenomena, significance, eventNumber, year, expires)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*VTECUGC
	for rows.Next() {
		ugc := &VTECUGC{}
		if err := ugc.scan(rows); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

// Database model for a VTEC UGC relation
type VTECUGC struct {
	ID           int        `json:"id"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at,omitempty"`
	WFO          string     `json:"wfo"`
	Phenomena    string     `json:"phenomena"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	UGC          int        `json:"ugc"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts,omitempty"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends,omitempty"`
	EndInitial   time.Time  `json:"end_initial,omitempty"`
	Action       string     `json:"action"`
	Year         int        `json:"year"`
}

func (ugc *VTECUGC) scan(rows pgx.Rows) error {
	return rows.Scan(
		&ugc.ID,
		&ugc.CreatedAt,
		&ugc.UpdatedAt,
		&ugc.WFO,
		&ugc.Phenomena,
		&ugc.Significance,
		&ugc.EventNumber,
		&ugc.UGC,
		&ugc.Issued,
		&ugc.Starts,
		&ugc.Expires,
		&ugc.Ends,
		&ugc.EndInitial,
		&ugc.Action,
		&ugc.Year,
	)
}

func (ugc *VTECUGC) Insert(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
		INSERT INTO vtec.ugcs(wfo, phenomena, significance, event_number, ugc, issued, starts, expires, ends, end_initial, action, year) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);
		`, ugc.WFO, ugc.Phenomena, ugc.Significance, ugc.EventNumber, ugc.UGC,
		ugc.Issued, ugc.Starts, ugc.Expires, ugc.Ends, ugc.EndInitial,
		ugc.Action, ugc.Year)
	if err != nil {
		return err
	}

	defer rows.Close()

	if rows.Next() {
		if err := ugc.scan(rows); err != nil {
			return err
		}
	}

	return nil
}

func BulkUpdateUGCsById(db *pgxpool.Pool, ugcs []int, expires time.Time, ends time.Time, action string, wfo string, phenomena string, significance string, eventNumber int, year int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
	UPDATE vtec.ugcs SET expires = $1, ends = $2, action = $3 WHERE
	wfo = $4 AND phenomena = $5 AND significance = $6 AND event_number = $7 AND year = $8
	AND ugc = ANY($9)
	`, expires, ends, action, wfo, phenomena, significance, eventNumber,
		year, ugcs)
	return err
}

func (ugc *VTECUGC) Delete(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
		DELETE FROM vtec.ugcs WHERE id = $1
		`, ugc.ID)
	return err
}
