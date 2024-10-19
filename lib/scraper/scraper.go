package scraper

// read-only scrapers are mostly stateless, each method is independent of each other,
// the output is dependent solely on the input.
// EXCEPT for the login state, that is an implied input for each method.

// mutatable scrapers are inherently stateful, they mutate state on the server. (thankfully we don't use many of these)

// each scraping method (for read-only) generally has this structure:
// 1. make assertions on input validity.
// 2. transform input into HTTP request object (method, headers, body)
// 3. make request.
// 4. make assertions on response validity. (expected body type, expected url, expected status, etc...)
// 5. transform HTTP response (url, body, headers) into output structure.

// this can generally be abstracted into 3 steps:
// 1) input -> req 2) req -> res 3) res -> output
// which probably could also be turned into an interface?

// the part in which you transform a response into an output is also generally declarable
// it is usually -> various goquery selectors into a struct or slices of structs
//               -> json -> struct
//               -> some other format

// the scraper part is then the code that guides the program through the acquiring of all
// the information you want in the representation you want.
// it is the thing that combines various scraping methods into one data model.
