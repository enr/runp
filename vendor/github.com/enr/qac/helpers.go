package qac

// Bool returns a pointer to v. Use it to set *bool fields in struct
// literals without a named intermediate variable:
//
//	qac.OutputAssertion{IsEmpty: qac.Bool(true)}
//	qac.FileAssertion{Exists: qac.Bool(false)}
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v. Use it to set *int fields in struct
// literals without a named intermediate variable:
//
//	qac.StatusAssertion{EqualsTo: qac.Int(0)}
//	qac.OutputAssertion{LineCount: qac.Int(5)}
func Int(v int) *int { return &v }
