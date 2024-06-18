import { combineReducers, configureStore } from '@reduxjs/toolkit';
import { persistReducer, persistStore } from 'redux-persist';
import storage from 'redux-persist/lib/storage';
import { authReducers } from './auth';
import { commonReducers } from './common';

const persistedReducer = combineReducers({
  auth: persistReducer({ key: "auth", storage, }, authReducers),
  common: commonReducers,
});

export const reduxStore = configureStore({
  reducer: persistedReducer,
});

export const reduxStorePersistor = persistStore(reduxStore);

export type RootState = ReturnType<typeof reduxStore.getState>;
export type AppDispatch = typeof reduxStore.dispatch;