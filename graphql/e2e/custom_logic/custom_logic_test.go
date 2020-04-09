/*
 * Copyright 2020 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package custom_logic

import (
	"testing"

	"github.com/dgraph-io/dgraph/graphql/e2e/common"
	"github.com/stretchr/testify/require"
)

const (
	alphaURL      = "http://localhost:8180/graphql"
	alphaAdminURL = "http://localhost:8180/admin"
	customTypes   = `type MovieDirector @remote {
		id: ID!
		name: String!
		directed: [Movie]
	}

	type Movie @remote {
		id: ID!
		name: String!
		director: [MovieDirector]
	}
	type Continent @remote {
		code: String
		name: String
		countries: [Country]
	  }
	  
	  type Country @remote {
		code: String
		name: String
		native: String
		phone: String
		continent: Continent
		currency: String
		languages: [Language]
		emoji: String
		emojiU: String
		states: [State]
	  }
	  
	  type Language @remote {
		code: String
		name: String
		native: String
		rtl: Int
	  }
	  
	  
	  type State @remote {
		code: String
		name: String
		country: Country
	  }
`
)

func updateSchema(t *testing.T, sch string) *common.GraphQLResponse {
	add := &common.GraphQLParams{
		Query: `mutation updateGQLSchema($sch: String!) {
			updateGQLSchema(input: { set: { schema: $sch }}) {
				gqlSchema {
					schema
				}
			}
		}`,
		Variables: map[string]interface{}{"sch": sch},
	}
	return add.ExecuteAsPost(t, alphaAdminURL)
}

func TestCustomGetQuery(t *testing.T) {
	schema := customTypes + `
	type Query {
        myFavoriteMovies(id: ID!, name: String!, num: Int): [Movie] @custom(http: {
                url: "http://mock:8888/favMovies/$id?name=$name&num=$num",
                method: "GET"
        })
	}`
	common.RequireNoGQLErrors(t, updateSchema(t, schema))
	query := `
	query {
		myFavoriteMovies(id: "0x123", name: "Author", num: 10) {
			id
			name
			director {
				id
				name
			}
		}
	}`
	params := &common.GraphQLParams{
		Query: query,
	}

	result := params.ExecuteAsPost(t, alphaURL)
	require.Nil(t, result.Errors)

	expected := `{"myFavoriteMovies":[{"id":"0x3","name":"Star Wars","director":[{"id":"0x4","name":"George Lucas"}]},{"id":"0x5","name":"Star Trek","director":[{"id":"0x6","name":"J.J. Abrams"}]}]}`
	require.JSONEq(t, expected, string(result.Data))
}

func TestCustomPostQuery(t *testing.T) {
	schema := customTypes + `
	type Query {
        myFavoriteMoviesPost(id: ID!, name: String!, num: Int): [Movie] @custom(http: {
                url: "http://mock:8888/favMoviesPost/$id?name=$name&num=$num",
                method: "POST"
        })
	}`
	common.RequireNoGQLErrors(t, updateSchema(t, schema))

	query := `
	query {
		myFavoriteMoviesPost(id: "0x123", name: "Author", num: 10) {
			id
			name
			director {
				id
				name
			}
		}
	}`
	params := &common.GraphQLParams{
		Query: query,
	}

	result := params.ExecuteAsPost(t, alphaURL)
	require.Nil(t, result.Errors)

	expected := `{"myFavoriteMoviesPost":[{"id":"0x3","name":"Star Wars","director":[{"id":"0x4","name":"George Lucas"}]},{"id":"0x5","name":"Star Trek","director":[{"id":"0x6","name":"J.J. Abrams"}]}]}`
	require.JSONEq(t, expected, string(result.Data))
}

func TestCustomQueryShouldForwardHeaders(t *testing.T) {
	schema := customTypes + `
	type Query {
        verifyHeaders(id: ID!): [Movie] @custom(http: {
                url: "http://mock:8888/verifyHeaders",
				method: "GET",
				forwardHeaders: ["X-App-Token", "X-User-Id"]
        })
	}`
	common.RequireNoGQLErrors(t, updateSchema(t, schema))

	query := `
	query {
		verifyHeaders(id: "0x123") {
			id
			name
		}
	}`
	params := &common.GraphQLParams{
		Query: query,
		Headers: map[string][]string{
			"X-App-Token":   []string{"app-token"},
			"X-User-Id":     []string{"123"},
			"Random-header": []string{"random"},
		},
	}

	result := params.ExecuteAsPost(t, alphaURL)
	require.Nil(t, result.Errors)
	expected := `{"verifyHeaders":[{"id":"0x3","name":"Star Wars"}]}`
	require.Equal(t, expected, string(result.Data))
}

func TestForInvalidCustomQuery(t *testing.T) {
	schema := customTypes + `
	type Query {
		getCountry(id: ID!): Country! @custom(http: {url: "http://mock:8888/noquery", method: "POST",forwardHeaders: ["Content-Type"]}, graphql: {query: "country(code: $id)"})
	}	
	`
	res := updateSchema(t, schema)
	require.Equal(t, res.Errors[0].Error(), "couldn't rewrite mutation updateGQLSchema because input:46: Type Query; Field getCountry; country is not present in remote schema\n")
}

func TestForInvalidArguement(t *testing.T) {
	schema := customTypes + `
	type Query {
		getCountry(id: ID!): Country! @custom(http: {url: "http://mock:8888/invalidargument", method: "POST",forwardHeaders: ["Content-Type"]}, graphql: {query: "country(code: $id)"})
	}	
	`
	res := updateSchema(t, schema)
	require.Equal(t, res.Errors[0].Error(), "couldn't rewrite mutation updateGQLSchema because input:46: Type Query; Field getCountry;code arg not present in the remote query country\n")
}

func TestForInvalidType(t *testing.T) {
	schema := customTypes + `
	type Query {
		getCountry(id: ID!): Country! @custom(http: {url: "http://mock:8888/invalidtype", method: "POST",forwardHeaders: ["Content-Type"]}, graphql: {query: "country(code: $id)"})
	}	
	`
	res := updateSchema(t, schema)
	require.Equal(t, res.Errors[0].Error(), "couldn't rewrite mutation updateGQLSchema because input:46: Type Query; Field getCountry; expected type for variable  $id is Int. But got ID!\n")
}

func TestCustomLogicGraphql(t *testing.T) {
	schema := customTypes + `
	type Query {
		getCountry(id: ID!): Country! @custom(http: {url: "http://mock:8888/validcountry", method: "POST"}, graphql: {query: "country(code: $id)"})
	}	
	`
	res := updateSchema(t, schema)
	require.Nil(t, res.Errors)
	query := `
	query {
		getCountry(id: "BI"){
			code
			name 
		}
	}`
	params := &common.GraphQLParams{
		Query: query,
	}

	result := params.ExecuteAsPost(t, alphaURL)
	common.RequireNoGQLErrors(t, result)
	require.JSONEq(t, string(result.Data), `
	{"getCountry":{"code":"BI","name":"Burundi"}}
	`)
}

func TestCustomLogicGraphqlWithError(t *testing.T) {
	schema := customTypes + `
	type Query {
		getCountry(id: ID!): Country! @custom(http: {url: "http://mock:8888/validcountrywitherror", method: "POST"}, graphql: {query: "country(code: $id)"})
	}	
	`
	common.RequireNoGQLErrors(t, updateSchema(t, schema))
	query := `
	query {
		getCountry(id: "BI"){
			code
			name 
		}
	}`
	params := &common.GraphQLParams{
		Query: query,
	}

	result := params.ExecuteAsPost(t, alphaURL)
	require.JSONEq(t, string(result.Data), `
	{"getCountry":{"code":"BI","name":"Burundi"}}
	`)
	require.Equal(t, "dummy error", result.Errors.Error())
}
