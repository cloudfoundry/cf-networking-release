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
	idStub   int
	typeStub string
	errStub  error
}

func (scanner *fakeScanner) Scan(dest ...interface{}) error {
	switch d := dest[0].(type) {
	case *int:
		*d = scanner.idStub
	default:
		panic("not an int")
	}

	if len(dest) > 1 {
		switch d := dest[1].(type) {
		case *string:
			if scanner.typeStub != "" {
				*d = scanner.typeStub
			} else {
				*d = "app"
			}
		default:
			panic("not an int")
		}
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
		var (
			recordFoundScanner *fakeScanner
			errorScanner       *fakeScanner

			firstBlankRowScanner *fakeScanner
		)

		BeforeEach(func() {
			recordFoundScanner = &fakeScanner{idStub: 1}
			errorScanner = &fakeScanner{idStub: 1, errStub: errors.New("some error")}

			firstBlankRowScanner = &fakeScanner{idStub: 5}

			fakeTx.QueryRowReturns(recordFoundScanner)
		})

		Context("when there already is a row with the provided guid", func() {
			It("returns the id of the row and no error", func() {
				id, err := groupTable.Create(fakeTx, "guid", "app")
				Expect(id).To(Equal(1))
				Expect(err).NotTo(HaveOccurred())
			})

			Context("but there is also an error", func() {
				BeforeEach(func() {
					fakeTx.QueryRowReturns(errorScanner)
				})

				It("returns -1 and an error", func() {
					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when there is not a row with the provided guid", func() {
			var (
				notFoundScanner *fakeScanner
			)

			BeforeEach(func() {
				notFoundScanner = &fakeScanner{idStub: -1, errStub: sql.ErrNoRows}

				fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
				fakeTx.QueryRowReturnsOnCall(1, firstBlankRowScanner)
				fakeTx.ExecReturnsOnCall(0, nil, nil)
			})

			It("finds a blank row and populates it with the given guid", func() {
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

			Context("and there are no empty rows that can be populated", func() {
				var (
					noBlankRowScanner *fakeScanner
				)

				BeforeEach(func() {
					noBlankRowScanner = &fakeScanner{idStub: 5, errStub: errors.New("no blank rows")}
					fakeTx.QueryRowReturnsOnCall(0, notFoundScanner)
					fakeTx.QueryRowReturnsOnCall(1, noBlankRowScanner)
				})

				It("returns -1 and an error", func() {
					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to find available tag"))
					Expect(err.Error()).To(ContainSubstring("no blank rows"))
				})
			})

			Context("and there is an error updating the row", func() {
				BeforeEach(func() {
					fakeTx.ExecReturnsOnCall(0, nil, errors.New("some error"))
				})

				It("returns -1 and an error", func() {
					id, err := groupTable.Create(fakeTx, "guid", "app")
					Expect(id).To(Equal(-1))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("some error"))
				})

				Context("but the error is for a duplicate entry", func() {
					var (
						idAndTypeRecordScanner *fakeScanner
					)

					BeforeEach(func() {
						idAndTypeRecordScanner = &fakeScanner{idStub: 1, typeStub: "app"}
						fakeTx.QueryRowReturnsOnCall(2, idAndTypeRecordScanner)
					})

					Context("and the duplicate row's groupType matches", func() {
						Context("and the driver is postgres", func() {
							BeforeEach(func() {
								fakeTx.DriverNameReturns("postgres")
								fakeTx.ExecReturnsOnCall(0, nil, errors.New("Postgres error 23505: duplicate entry"))
							})

							It("does not update the row, returns the original row, and does not error", func() {
								id, err := groupTable.Create(fakeTx, "guid", "app")
								Expect(id).To(Equal(1))
								Expect(err).ToNot(HaveOccurred())
							})

							Context("and the duplicate row's groupType doesn't match", func() {
								BeforeEach(func() {
									idAndTypeRecordScanner = &fakeScanner{idStub: 1, typeStub: "meow"}
									fakeTx.QueryRowReturnsOnCall(2, idAndTypeRecordScanner)
								})

								It("returns an error", func() {
									id, err := groupTable.Create(fakeTx, "guid", "app")
									Expect(id).To(Equal(-1))
									Expect(err).To(HaveOccurred())
								})
							})
						})

						Context("and the driver is mysql", func() {
							BeforeEach(func() {
								fakeTx.DriverNameReturns("mysql")
								fakeTx.ExecReturnsOnCall(0, nil, errors.New("Mysql error 1062: duplicate entry"))
							})

							It("does not update the row, returns the original row, and does not error", func() {
								id, err := groupTable.Create(fakeTx, "guid", "app")
								Expect(id).To(Equal(1))
								Expect(err).ToNot(HaveOccurred())
							})

							Context("and the duplicate row's groupType doesn't match", func() {
								BeforeEach(func() {
									idAndTypeRecordScanner = &fakeScanner{idStub: 1, typeStub: "meow"}
									fakeTx.QueryRowReturnsOnCall(2, idAndTypeRecordScanner)
								})

								It("returns an error", func() {
									id, err := groupTable.Create(fakeTx, "guid", "app")
									Expect(id).To(Equal(-1))
									Expect(err).To(HaveOccurred())
								})
							})
						})
					})
				})
			})
		})
	})
})
