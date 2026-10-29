package repository

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreDBConfig_DefaultsUseMainDatabaseAndMStore(t *testing.T) {
	c := LoadStoreDBConfig(viper.New())

	assert.False(t, c.Separate())
	require.NoError(t, c.Validate())
	assert.Equal(t, "SELECT TOP (2000) StoreCode AS StoreCode, StoreName AS StoreName, Latitude AS Latitude, Longitude AS Longitude"+
		" FROM AMFIBI_MASTER.dbo.MStore WHERE Latitude BETWEEN ? AND ? AND Longitude BETWEEN ? AND ? AND StoreName <> 'FOODTRUCK 1' AND IsActive = 1",
		c.Query(2000))
	assert.Equal(t, 1433, c.Port)
}

func TestStoreDBConfig_SeparateDatabaseWithOwnColumns(t *testing.T) {
	v := viper.New()
	v.Set("store_database.driver", "MySQL")
	v.Set("store_database.host", "10.0.0.5")
	v.Set("store_database.database", "stores")
	v.Set("store_database.table", "store_master")
	v.Set("store_database.code_column", "kode_toko")
	v.Set("store_database.name_column", "nama_toko")
	v.Set("store_database.lat_column", "lat")
	v.Set("store_database.lng_column", "lng")
	v.Set("store_database.address_column", "alamat")
	v.Set("store_database.active_column", "")
	c := LoadStoreDBConfig(v)

	assert.True(t, c.Separate())
	assert.Equal(t, 3306, c.Port)
	require.NoError(t, c.Validate())
	assert.Equal(t, "SELECT kode_toko AS StoreCode, nama_toko AS StoreName, lat AS Latitude, lng AS Longitude, alamat AS Address"+
		" FROM store_master WHERE lat BETWEEN ? AND ? AND lng BETWEEN ? AND ? AND nama_toko <> 'FOODTRUCK 1' LIMIT 50",
		c.Query(50))
}

func TestStoreDBConfig_RefusesNonNames(t *testing.T) {
	for _, tc := range []struct{ key, val string }{
		{"table", "MStore; DROP TABLE Sites"},
		{"table", "a.b.c.d"},
		{"code_column", "StoreCode--"},
		{"address_column", "[Address]"},
		{"active_column", "1=1 OR x"},
		{"driver", "postgres"},
	} {
		t.Run(tc.key+" "+tc.val, func(t *testing.T) {
			v := viper.New()
			v.Set("store_database."+tc.key, tc.val)
			err := LoadStoreDBConfig(v).Validate()
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrStoreDBConfig))
		})
	}
}

func TestStoreDBConfig_CacheTagFollowsSourceNotPassword(t *testing.T) {
	a := viper.New()
	b := viper.New()
	b.Set("store_database.password", "something-else")
	c := viper.New()
	c.Set("store_database.table", "OtherDB.dbo.MStore")

	assert.Equal(t, LoadStoreDBConfig(a).tag(), LoadStoreDBConfig(b).tag())
	assert.NotEqual(t, LoadStoreDBConfig(a).tag(), LoadStoreDBConfig(c).tag())
}

func TestRedactRemovesPassword(t *testing.T) {
	assert.Equal(t, "login failed for *** at host", redact(errors.New("login failed for pw123 at host"), "pw123"))
}
