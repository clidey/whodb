import { Route, Routes } from "react-router-dom";
import './app.css';
import { Notifications } from './components/notifications';
import { InternalRoutes, PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { map } from "lodash";

export const App = () => {
  
  return (
    <div className="h-[100vh] w-[100vw]">
      <Notifications />
      <Routes>
        <Route path={InternalRoutes.Dashboard.path} element={<PrivateRoute />}>
          {map(getRoutes(), route => (
            <Route path={route.path} element={route.component} />
          ))}
        </Route>
        <Route path={PublicRoutes.Login.path} element={PublicRoutes.Login.component} />
      </Routes>
    </div>
  );
}

export default App;
