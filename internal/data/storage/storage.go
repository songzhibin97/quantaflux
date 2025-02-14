package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/songzhibin97/quantaflux/internal/models"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &PostgresStorage{db: db}

	err = s.initTables()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return s, nil
}

// SaveTokenInfo implements DataStorage interface
func (s *PostgresStorage) SaveTokenInfo(ctx context.Context, info *models.TokenInfo) error {
	query := `
        INSERT INTO token_info (
            symbol, name, contract_address, network, launch_type,
            initial_price, total_supply, circulating_supply,
            team_allocation, vesting_schedule, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11
        )
        ON CONFLICT (symbol) DO UPDATE SET
            name = EXCLUDED.name,
            contract_address = EXCLUDED.contract_address,
            network = EXCLUDED.network,
            launch_type = EXCLUDED.launch_type,
            initial_price = EXCLUDED.initial_price,
            total_supply = EXCLUDED.total_supply,
            circulating_supply = EXCLUDED.circulating_supply,
            team_allocation = EXCLUDED.team_allocation,
            vesting_schedule = EXCLUDED.vesting_schedule,
            updated_at = EXCLUDED.updated_at
    `

	_, err := s.db.ExecContext(ctx, query,
		info.Symbol,
		info.Name,
		info.ContractAddress,
		info.Network,
		info.LaunchType,
		info.InitialPrice,
		info.TotalSupply,
		info.CirculatingSupply,
		info.TeamAllocation,
		info.VestingSchedule,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save token info: %w", err)
	}

	return nil
}

// SaveMarketData implements DataStorage interface
func (s *PostgresStorage) SaveMarketData(ctx context.Context, data *models.MarketData) error {
	query := `
        INSERT INTO market_data (
            symbol, price, volume_24h, market_cap,
            price_change_1h, price_change_24h, timestamp
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        )
    `

	_, err := s.db.ExecContext(ctx, query,
		data.Symbol,
		data.Price,
		data.Volume24h,
		data.MarketCap,
		data.PriceChange1h,
		data.PriceChange24h,
		data.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save market data: %w", err)
	}

	return nil
}

// GetHistoricalData implements DataStorage interface
func (s *PostgresStorage) GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]models.MarketData, error) {
	query := `
        SELECT symbol, price, volume_24h, market_cap,
               price_change_1h, price_change_24h, timestamp
        FROM market_data
        WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
        ORDER BY timestamp ASC
    `

	rows, err := s.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical data: %w", err)
	}
	defer rows.Close()

	var result []models.MarketData
	for rows.Next() {
		var data models.MarketData
		err := rows.Scan(
			&data.Symbol,
			&data.Price,
			&data.Volume24h,
			&data.MarketCap,
			&data.PriceChange1h,
			&data.PriceChange24h,
			&data.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", err)
		}
		result = append(result, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating market data rows: %w", err)
	}

	return result, nil
}

// GetProjectMetrics implements DataStorage interface
func (s *PostgresStorage) GetProjectMetrics(ctx context.Context, symbol string) (*models.ProjectMetrics, error) {
	query := `
        SELECT m.token_info_id, m.social_score, m.development_score,
               m.community_growth, m.market_sentiment, m.risk_score,
               m.updated_at, t.symbol, t.name, t.contract_address,
               t.network, t.launch_type, t.initial_price, t.total_supply,
               t.circulating_supply, t.team_allocation, t.vesting_schedule
        FROM project_metrics m
        JOIN token_info t ON m.token_info_id = t.id
        WHERE t.symbol = $1
        ORDER BY m.updated_at DESC
        LIMIT 1
    `

	var metrics models.ProjectMetrics
	var tokenInfo models.TokenInfo

	err := s.db.QueryRowContext(ctx, query, symbol).Scan(
		&metrics.TokenInfo.Symbol,
		&metrics.SocialScore,
		&metrics.DevelopmentScore,
		&metrics.CommunityGrowth,
		&metrics.MarketSentiment,
		&metrics.RiskScore,
		&tokenInfo.Symbol,
		&tokenInfo.Name,
		&tokenInfo.ContractAddress,
		&tokenInfo.Network,
		&tokenInfo.LaunchType,
		&tokenInfo.InitialPrice,
		&tokenInfo.TotalSupply,
		&tokenInfo.CirculatingSupply,
		&tokenInfo.TeamAllocation,
		&tokenInfo.VestingSchedule,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("no metrics found for symbol: %s", symbol)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project metrics: %w", err)
	}

	metrics.TokenInfo = tokenInfo
	return &metrics, nil
}

func (s *PostgresStorage) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS token_info (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(50) UNIQUE NOT NULL,
			name VARCHAR(100),
			contract_address VARCHAR(100),
			network VARCHAR(50),
			launch_type VARCHAR(50),
			initial_price NUMERIC(18, 8),
			total_supply NUMERIC(18, 8),
			circulating_supply NUMERIC(18, 8),
			team_allocation NUMERIC(18, 8),
			vesting_schedule TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS market_data (
			id SERIAL PRIMARY KEY,
			symbol VARCHAR(50) NOT NULL,
			price NUMERIC(18, 8),
			volume_24h NUMERIC(18, 8),
			market_cap NUMERIC(18, 8),
			price_change_1h NUMERIC(10, 4),
			price_change_24h NUMERIC(10, 4),
			timestamp TIMESTAMP NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS project_metrics (
			id SERIAL PRIMARY KEY,
			token_info_id INT NOT NULL,
			social_score NUMERIC(10, 4),
			development_score NUMERIC(10, 4),
			community_growth NUMERIC(10, 4),
			market_sentiment NUMERIC(10, 4),
			risk_score NUMERIC(10, 4),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}
	return nil
}
