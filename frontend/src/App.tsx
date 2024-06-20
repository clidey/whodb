import { map } from "lodash";
import { Navigate, Route, Routes } from "react-router-dom";
import './app.css';
import { Notifications } from './components/notifications';
import { InternalRoutes, PrivateRoute, PublicRoutes, getRoutes } from './config/routes';

export const App = () => {
  return (
    <div className="h-[100vh] w-[100vw]">
      <Notifications />
      <Routes>
        <Route path="/" element={<PrivateRoute />}>
          {map(getRoutes(), route => (
            <Route key={route.path} path={route.path} element={route.component} />
          ))}
          <Route path="/" element={<Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />} />
        </Route>
        <Route path={PublicRoutes.Login.path} element={PublicRoutes.Login.component} />
      </Routes>
    </div>
  );
}

export default App;
