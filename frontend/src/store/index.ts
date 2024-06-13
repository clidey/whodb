import { configureStore } from '@reduxjs/toolkit'
import { authReducers } from './auth'
import { commonReducers } from './common'

export const reduxStore = configureStore({
  reducer: {
    auth: authReducers,
    common: commonReducers,
  },
})


export type RootState = ReturnType<typeof reduxStore.getState>
export type AppDispatch = typeof reduxStore.dispatch