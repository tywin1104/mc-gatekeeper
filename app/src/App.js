import React from 'react';
import './App.css';
import {
  BrowserRouter as Router,
  Switch,
  Route,
  Link
} from "react-router-dom";
import Application from './components/Application'

function App() {
  return (
    <Router>
      <div>
        <nav>
          <ul>
            <li>
              <Link to="/">Application</Link>
            </li>
            <li>
              <Link to="/about">About</Link>
            </li>
            <li>
              <Link to="/users">Users</Link>
            </li>
          </ul>
        </nav>

        <Switch>
          <Route path="/about">
            {/* <About /> */}
            <h1>first route</h1>
          </Route>
          <Route path="/users">
            {/* <Users /> */}
            <h1>second route</h1>
          </Route>
          <Route path="/">
            <Application  ></Application>
          </Route>
        </Switch>
      </div>
    </Router>
  );
}

export default App;
