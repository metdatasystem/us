package warnings

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twpayne/go-geos"
)

type Warning struct {
	ID            int        `json:"id"`
	CreatedAt     time.Time  `json:"created_at,omitzero"`
	UpdatedAt     time.Time  `json:"updated_at,omitzero"`
	Issued        time.Time  `json:"issued"`
	Starts        *time.Time `json:"starts,omitzero"`
	Expires       time.Time  `json:"expires"`
	Ends          time.Time  `json:"ends,omitzero"`
	EndInitial    time.Time  `json:"end_initial,omitzero"`
	Text          string     `json:"text"`
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

func GetWarning(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int) (*Warning, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM warnings.warnings WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
	`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var warning *Warning
	if rows.Next() {
		warning = &Warning{}
		if err := warning.scan(rows); err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	return warning, nil
}

func (warning *Warning) scan(row pgx.Rows) error {
	return row.Scan(
		&warning.ID,
		&warning.CreatedAt,
		&warning.UpdatedAt,
		&warning.Issued,
		&warning.Starts,
		&warning.Expires,
		&warning.Ends,
		&warning.EndInitial,
		&warning.Text,
		&warning.WFO,
		&warning.Action,
		&warning.Class,
		&warning.Phenomena,
		&warning.Significance,
		&warning.EventNumber,
		&warning.Year,
		&warning.Title,
		&warning.IsEmergency,
		&warning.IsPDS,
		&warning.Geom,
		&warning.Direction,
		&warning.Location,
		&warning.Speed,
		&warning.SpeedText,
		&warning.TMLTime,
		&warning.UGC,
		&warning.Tornado,
		&warning.Damage,
		&warning.HailThreat,
		&warning.HailTag,
		&warning.WindThreat,
		&warning.WindTag,
		&warning.FlashFlood,
		&warning.RainfallTag,
		&warning.FloodTagDam,
		&warning.SpoutTag,
		&warning.SnowSquall,
		&warning.SnowSquallTag,
	)
}

func (warning *Warning) Insert(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
       INSERT INTO warnings.warnings(
	       issued, starts, expires, ends, end_initial, text, 
		   wfo, action, class, phenomena, significance, event_number, year, 
		   title, is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
		   ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
		   rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag
       ) VALUES (
	       $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 
		   $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35
       )
       `,
		warning.Issued,
		warning.Starts,
		warning.Expires,
		warning.Ends,
		warning.EndInitial,
		warning.Text,
		warning.WFO,
		warning.Action,
		warning.Class,
		warning.Phenomena,
		warning.Significance,
		warning.EventNumber,
		warning.Year,
		warning.Title,
		warning.IsEmergency,
		warning.IsPDS,
		warning.Geom,
		warning.Direction,
		warning.Location,
		warning.Speed,
		warning.SpeedText,
		warning.TMLTime,
		warning.UGC,
		warning.Tornado,
		warning.Damage,
		warning.HailThreat,
		warning.HailTag,
		warning.WindThreat,
		warning.WindTag,
		warning.FlashFlood,
		warning.RainfallTag,
		warning.FloodTagDam,
		warning.SpoutTag,
		warning.SnowSquall,
		warning.SnowSquallTag,
	)
	if err != nil {
		return err
	}

	if rows.Next() {
		if err := warning.scan(rows); err != nil {
			return err
		}
	}

	return nil
}

func (warning *Warning) Update(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
		UPDATE warnings.warnings SET updated_at = $1, expires = $2, ends = $3, text = $4, action = $5, title = $6, 
    	is_emergency = $7, is_pds = $8, geom = $9, direction = $10, location = $11, speed = $12, 
    	speed_text = $13, tml_time = $14, ugc = $15, tornado = $16, damage = $17, hail_threat = $18, 
    	hail_tag = $19, wind_threat = $20, wind_tag = $21, flash_flood = $22, rainfall_tag = $23, 
    	flood_tag_dam = $24, spout_tag = $25, snow_squall = $26, snow_squall_tag = $27
		WHERE wfo = $28 AND phenomena = $29 AND significance = $30 AND event_number = $31 AND year = $32
		`, time.Now().UTC(), warning.Expires, warning.Ends, warning.Text, warning.Action, warning.Title,
		warning.IsEmergency, warning.IsPDS, warning.Geom, warning.Direction, warning.Location, warning.Speed,
		warning.SpeedText, warning.TMLTime, warning.UGC, warning.Tornado, warning.Damage, warning.HailThreat, warning.HailTag,
		warning.WindThreat, warning.WindTag, warning.FlashFlood, warning.RainfallTag, warning.FloodTagDam,
		warning.SpoutTag, warning.SnowSquall, warning.SnowSquallTag, warning.WFO, warning.Phenomena,
		warning.Significance, warning.EventNumber, warning.Year)
	if err != nil {
		return err
	}

	return nil
}
