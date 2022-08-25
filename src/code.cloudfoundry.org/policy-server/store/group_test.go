package store_test

import (
	"database/sql"
	"errors"

	"code.cloudfoundry.org/cf-networking-helpers/db/fakes"
	"code.cloudfoundry.org/policy-server/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeScanner struct {
	indexStub int
	errStub   error
}

func (scanner *fakeScanner) Scan(dest ...interface{}) error {
	switch d := dest[0].(type) {
	case *int:
		*d = scanner.indexStub
	default:
		panic("not an int")
	}

	return scanner.errStub
}

var _ = Describe("GroupTable", func() {
	var (
		groupTable store.GroupTable
		fakeTx     *fakes.Transaction
	)

	BeforeEach(func() {
		groupTable = store.GroupTable{}
		fakeTx = &fakes.Transaction{}
		fakeTx.RebindStub = func(input string) string {
			return input
		}
	})

	Describe("Create", func() {
		Context("when there already is a row with the provided guid", func() {
			It("returns the id of the row and no error", func() {
				fakeTx.QueryRowReturns(&fakeScanner{indexStub: 1})

				id, err := groupTable.Create(fakeTx, "guid", "app")
				Expect(id).To(Equal(1))
				Expect(err).NotTo(HaveOccurred())
			})

			Context("but there is also an error", func() {
				It("returns -1 and an error", func() {
					fakeTx.QueryRowReturns(&fakeScanner{indexStub: 1, errStub: errors.New("some error")})

					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when there is not a row with the provided guid", func() {
			It("finds a blank row and populates it with the given guid", func() {
				notFoundScanner := &fakeScanner{indexStub: -1, errStub: sql.ErrNoRows}
				firstBlankRowScanner := &fakeScanner{indexStub: 5}
				fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
				fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)
				fakeTx.ExecReturnsOnCall(0, nil, nil)

				id, err := groupTable.Create(fakeTx, "guid", "app")
				Expect(id).To(Equal(5))
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeTx.QueryRowCallCount()).To(Equal(2))
				Expect(fakeTx.QueryRowArgsForCall(1)).To(Equal(`SELECT id FROM "groups"
		WHERE guid is NULL
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`))

				Expect(fakeTx.ExecCallCount()).To(Equal(1))
				queryString, args := fakeTx.ExecArgsForCall(0)
				Expect(queryString).To(Equal(`
			UPDATE "groups" SET guid = ?, type =  ?
			WHERE id = ?
		`))
				Expect(args).To(HaveLen(3))
				guid, ok := args[0].(string)
				Expect(ok).To(BeTrue())

				typeStr, ok := args[1].(string)
				Expect(ok).To(BeTrue())

				savedID, ok := args[2].(int)
				Expect(ok).To(BeTrue())

				Expect(guid).To(Equal("guid"))
				Expect(typeStr).To(Equal("app"))
				Expect(savedID).To(Equal(5))
			})

			Context("when there are no empty rows that can be populated", func() {
				It("returns -1 and an error", func() {
					notFoundScanner := &fakeScanner{indexStub: -1, errStub: sql.ErrNoRows}
					firstBlankRowScanner := &fakeScanner{indexStub: 5, errStub: errors.New("no blank rows")}
					fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
					fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)

					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to find available tag"))
					Expect(err.Error()).To(ContainSubstring("no blank rows"))
				})
			})

			Context("when there is an error updating the row", func() {
				It("returns -1 and an error", func() {
					notFoundScanner := &fakeScanner{indexStub: -1, errStub: sql.ErrNoRows}
					firstBlankRowScanner := &fakeScanner{indexStub: 5}
					fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
					fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)
					fakeTx.ExecReturnsOnCall(0, nil, errors.New("some error"))

					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("some error"))
				})

				Context("when the error is for a duplicate entry", func() {
					Context("and the driver is postgres", func() {
						It("does not update the row, returns the original row, and does not error", func() {
							notFoundScanner := &fakeScanner{indexStub: -1, errStub: sql.ErrNoRows}
							firstBlankRowScanner := &fakeScanner{indexStub: 5}
							foundRowScanner := &fakeScanner{indexStub: 1}
							fakeTx.DriverNameReturns("postgres")
							fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
							fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)
							fakeTx.QueryRowReturnsOnCall(2, foundRowScanner)
							fakeTx.ExecReturnsOnCall(0, nil, errors.New("Postgres error 23505: duplicate entry"))

							id, err := groupTable.Create(fakeTx, "guid", "app")
							Expect(id).To(Equal(1))
							Expect(err).ToNot(HaveOccurred())
						})
					})

					Context("and the driver is mysql", func() {
						It("does not update the row, returns the original row, and does not error", func() {
							notFoundScanner := &fakeScanner{indexStub: -1, errStub: sql.ErrNoRows}
							firstBlankRowScanner := &fakeScanner{indexStub: 5}
							foundRowScanner := &fakeScanner{indexStub: 1}
							fakeTx.DriverNameReturns("mysql")
							fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
							fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)
							fakeTx.QueryRowReturnsOnCall(2, foundRowScanner)
							fakeTx.ExecReturnsOnCall(0, nil, errors.New("Mysql error 1062: duplicate entry"))

							id, err := groupTable.Create(fakeTx, "guid", "app")
							Expect(id).To(Equal(1))
							Expect(err).ToNot(HaveOccurred())
						})
					})
				})
			})
		})
	})
})
