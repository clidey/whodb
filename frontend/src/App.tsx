import { map } from "lodash";
import { Route, Routes } from "react-router-dom";
import './app.css';
import { Notifications } from './components/notifications';
import { PrivateRoute, PublicRoutes, getRoutes } from './config/routes';

export const App = () => {
  return (
    <div className="h-[100vh] w-[100vw]">
      <Notifications />
      <Routes>
        <Route path="/" element={<PrivateRoute />}>
          {map(getRoutes(), route => (
            <Route key={route.path} path={route.path} element={route.component} />
          ))}
        </Route>
        <Route path={PublicRoutes.Login.path} element={PublicRoutes.Login.component} />
      </Routes>
    </div>
  );
}

export default App;
