import { ApolloClient, InMemoryCache, createHttpLink } from '@apollo/client';

let uri = "/api/query"
if (window.location.port === "3000") {
  uri = "http://localhost:8080/api/query";
}

const httpLink = createHttpLink({
  uri,
  credentials: "include",
});

export const graphqlClient = new ApolloClient({
  link: httpLink,
  cache: new InMemoryCache(),
  defaultOptions: {
      query: {
        fetchPolicy: "no-cache",
      },
      mutate: {
        fetchPolicy: "no-cache",
      },
  }
});
