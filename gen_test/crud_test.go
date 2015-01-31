package gen_test

import . "gopkg.in/check.v1"
import "testing"
import "database/sql"
import "github.com/hydrogen18/sillyquill/gen_test/dal"
import "github.com/hydrogen18/sillyquill/rt"
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

func (s *TestSuite) TestErrOnNonUniquelyIdentifiables(c *C) {
	//This type can never be identified uniquely
	i := new(dal.NotUniquelyIdentifiable)
	i.SetAge(42)
	i.SetId(44)

	err := i.FindOrCreate(s.db)
	c.Assert(err, FitsTypeOf, sillyquill_rt.RowNotUniquelyIdentifiableError{})

	//This instance can not be identified uniquely
	j := new(dal.Incident)
	resolution := "MEOW"
	j.SetResolution(&resolution)
	err = j.FindOrCreate(s.db)
	c.Assert(err, FitsTypeOf, sillyquill_rt.RowNotUniquelyIdentifiableError{})
}

func (s *TestSuite) TestNoOverwritingExistingFields(c *C) {
	i := new(dal.Incident)
	var resolution string
	resolution = "PEBKAC"
	i.SetResolution(&resolution)

	err := i.Create(s.db)
	c.Assert(err, IsNil)
	c.Assert(i.IsLoaded.Id, Equals, true)

	j := new(dal.Incident)
	j.SetId(i.Id)
	var notTheResolution string
	notTheResolution = "fatality"
	j.SetResolution(&notTheResolution)
	err = j.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	c.Assert(j.IsLoaded.Resolution, Equals, true)
	c.Assert(*j.Resolution, Equals, *i.Resolution)

}

func (s *TestSuite) TestNumeric(c *C) {
	aNumber := new(dal.Number)
	var v sillyquill_rt.Numeric
	v.SetString("2632624.626332")
	aNumber.SetValue(v)

	err := aNumber.Create(s.db)
	c.Assert(err, IsNil)
	c.Assert(aNumber.IsLoaded.Id, Equals, true)

	sameNumber := new(dal.Number)
	sameNumber.SetId(aNumber.Id)
	err = sameNumber.Get(s.db)
	c.Assert(err, IsNil)
	c.Assert(sameNumber.Value, DeepEquals, aNumber.Value)
}

func (s *TestSuite) TestNumericNull(c *C) {
	nullNumber := new(dal.NullNumber)
	nullNumber.SetTitle("mewo")
	err := nullNumber.Create(s.db)
	c.Assert(err, IsNil)

	aNumber := new(dal.NullNumber)
	var v sillyquill_rt.NullNumeric
	aNumber.SetTitle("kitties")
	v.SetString("135135.16136")
	aNumber.SetValue(&v)

	err = aNumber.Create(s.db)
	c.Assert(err, IsNil)
	c.Assert(aNumber.IsLoaded.Id, Equals, true)

	sameNumber := new(dal.NullNumber)
	sameNumber.SetId(aNumber.Id)
	err = sameNumber.Get(s.db)
	c.Assert(err, IsNil)
	c.Assert(sameNumber.Value, DeepEquals, aNumber.Value)

	sameNumber = new(dal.NullNumber)
	sameNumber.SetId(nullNumber.Id)
	err = sameNumber.Get(s.db)
	c.Assert(err, IsNil)
	c.Assert(sameNumber.Value, IsNil)
}

func (s *TestSuite) TestArchiveFiles(c *C) {
	aFile := new(dal.ArchiveFile)
	aFile.SetName("foo.txt")
	var FOO_DATA = []byte{0x1, 0x2, 0x3}
	aFile.SetData(FOO_DATA)
	err := aFile.Create(s.db)
	c.Assert(err, IsNil)
	fooId := aFile.Id

	aFile = new(dal.ArchiveFile)
	aFile.SetName("bar.txt")
	aFile.SetData([]byte{}) //Test that zero-length doesn't violate not-null constraint
	err = aFile.Create(s.db)
	c.Assert(err, IsNil)

	//Test load by unique
	aFile = new(dal.ArchiveFile)
	aFile.SetId(fooId)
	err = aFile.Get(s.db)
	c.Assert(err, IsNil)
	c.Assert(aFile.Name, Equals, "foo.txt")
	c.Assert(aFile.Data, DeepEquals, FOO_DATA)
}

func (s *TestSuite) TestPizzaDeliveryGuys(c *C) {
	aGuy := new(dal.PizzaDeliveryGuy)
	aGuy.SetName("bob")
	aGuy.SetGasMileage(16.4)
	err := aGuy.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	/** TODO fixme
	err = aGuy.FindOrCreate(s.db)
	c.Assert(err, Equals,NoColumnsSetError)
	**/
	//Test Reload
	err = aGuy.Reload(s.db)
	c.Assert(err, IsNil)

	//Test find by primary key
	sameGuy := new(dal.PizzaDeliveryGuy)
	sameGuy.SetName(aGuy.Name)
	err = sameGuy.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	//Create another pizza delivery guy
	secondGuy := new(dal.PizzaDeliveryGuy)
	secondGuy.SetName("rufus")
	secondGuy.SetGasMileage(36.0)
	err = secondGuy.FindOrCreate(s.db)
	c.Assert(err, IsNil)

	//Test Save
	aGuy.SetGasMileage(15.0)
	err = aGuy.Save(s.db)
	c.Assert(err, IsNil)

	//Test save w/ no params
	//TODO fixme
	/**
	err = aGuy.Save(s.db)
	c.Assert(err, IsNil)
	**/

	err = aGuy.Reload(s.db)
	c.Assert(err, IsNil)

	err = secondGuy.Reload(s.db)
	c.Assert(err, IsNil)

	//Test for wild where clause in update
	c.Assert(aGuy.GasMileage, Equals, 15.0)
	c.Assert(aGuy.GasMileage, Not(Equals), secondGuy.GasMileage)

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
