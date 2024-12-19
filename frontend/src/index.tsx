import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import { ApolloProvider } from "@apollo/client";
import { graphqlClient } from './config/graphql-client';
import { Provider } from "react-redux";
import { reduxStore, reduxStorePersistor } from './store';
import { App } from './app';
import { BrowserRouter } from "react-router-dom";
import 'reactflow/dist/style.css';
import { PersistGate } from 'redux-persist/integration/react';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);
root.render(
  <React.StrictMode>
    <BrowserRouter>
      <ApolloProvider client={graphqlClient}>
        <Provider store={reduxStore}>
          <PersistGate loading={null} persistor={reduxStorePersistor}>
            <App />
          </PersistGate>
        </Provider>
      </ApolloProvider>
    </BrowserRouter>
  </React.StrictMode>
);

