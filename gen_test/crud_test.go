package gen_test

import . "gopkg.in/check.v1"
import "testing"
import "database/sql"
import "github.com/hydrogen18/sillyquill/gen_test/dal"
import _ "github.com/lib/pq"
import "os"
import "time"

type TestSuite struct {
	db *sql.DB
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&TestSuite{})

func (s *TestSuite) SetUpSuite(c *C) {
	var err error
	s.db, err = sql.Open("postgres", os.Getenv("DB"))
	c.Assert(err, IsNil)
	err = s.db.Ping()
	c.Assert(err, IsNil)
}
func (s *TestSuite) TearDownSuite(c *C) {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *TestSuite) TestCreateCar(c *C) {
	aCar := new(dal.Car)
	aCar.SetMake("kia")
	aCar.SetModel("rio")
	aCar.SetPassengers(5)
	err := aCar.Create(s.db)
	c.Assert(err, IsNil)
	c.Assert(aCar.IsLoaded.UpdatedAt, Equals, true)
	c.Assert(aCar.UpdatedAt, Not(Equals), time.Time{})

	//Test searching by primary key
	sameCar := new(dal.Car)
	sameCar.SetMake(aCar.Make)
	sameCar.SetModel(aCar.Model)
	err = sameCar.Get(s.db)
	c.Assert(err, IsNil)

	//Get loads all columns by default
	c.Assert(sameCar.IsLoaded.Id, Equals, true)
	c.Assert(sameCar.Id, Equals, aCar.Id)

	//Test searching by unique column partial load
	sameCar = new(dal.Car)
	sameCar.SetId(aCar.Id)
	err = sameCar.Get(s.db, dal.Cars.Passengers)
	c.Assert(err, IsNil)
	c.Assert(sameCar.IsLoaded.Passengers, Equals, true)
	c.Assert(sameCar.IsLoaded.Make, Equals, false)
	c.Assert(sameCar.IsLoaded.Model, Equals, false)
	c.Assert(sameCar.Passengers, Equals, aCar.Passengers)

	//Create another car
	aCar = new(dal.Car)
	aCar.SetMake("mazda")
	aCar.SetModel("rx-7")
	aCar.SetPassengers(5)
	err = aCar.FindOrCreate(s.db)
	c.Assert(err, IsNil)
	c.Assert(aCar.Id, Not(Equals), sameCar.Id)

}

func (s *TestSuite) TestCreateTruck(c *C) {
	aTruck := new(dal.Truck)
	aTruck.SetMake("volvo")
	aTruck.SetModel("t-1000")
	aTruck.SetTonnage(13.5)
	err := aTruck.Create(s.db)
	c.Assert(err, IsNil)

	aTruck = new(dal.Truck)
	aTruck.SetMake("ford")
	aTruck.SetModel("f150")
	aTruck.SetTonnage(0.5)
	err = aTruck.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	aTruck = new(dal.Truck)
	aTruck.SetMake("chevy")
	aTruck.SetModel("k1500")
	aTruck.SetTonnage(0.5)
	now := time.Now().Truncate(time.Second).Add(10 * time.Minute)
	aTruck.SetCreatedAt(now)
	aTruck.SetUpdatedAt(now)
	err = aTruck.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	c.Assert(aTruck.CreatedAt.UTC(), DeepEquals, now.UTC())
	c.Assert(aTruck.UpdatedAt.UTC(), DeepEquals, now.UTC())

	sameTruck := new(dal.Truck)
	sameTruck.SetId(aTruck.Id)
	err = sameTruck.Get(s.db)
	c.Assert(err, IsNil)
	sameTruck.IsSet = aTruck.IsSet //Clear flags
	c.Assert(*sameTruck, Equals, *aTruck)
}
