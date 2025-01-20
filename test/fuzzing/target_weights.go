package fuzzing

// steps:
// - GetWeights(courseId, categories)
//   - courseId can either be:
//     - an id that matches a random existing row (95%)
//     - an id that doesn't exist (5%)
//   - categories is a list of random length (including 0) with:
//     - categories that exist (95%)
//     - categories that don't exist (5%)
// - AddCourse(courseId, courseName)
//   - courseId can either be:
//     - an id that matches an existing row (80%)
//     - an id that doesn't exist (20%)
//   - courseName will be a random string
// - add ps_account with a given accountId
// - fault inject: time passing by faster than usual / reversing
// - fault inject: failed db query (TODO)

// properties of the system:
// - making a snapshot should take no more than 10ms (on the p95, so no more than 5%
//     of MakeSnapshot should be more than 10ms)
// - getting a snapshot should take no more than 10ms (on the p95, so no more than 5%
//     of GetSnapshot should be more than 10ms)
// - a snapshot added to an account should be added to that account and no others

// randomly generate steps to try and find a situation where a property of the system is invalidated
