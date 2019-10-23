import React from 'react';
import './App.css';
import {
  BrowserRouter as Router,
  Switch,
  Route,
  Link
} from "react-router-dom";
import Application from './components/Application'
import CheckStatus from './components/CheckStatus'
import AdminAction from './components/AdminAction'

function App() {
  return (
    <Router>
      <div>
        <nav>
          <ul>
            <li>
              <Link to="/">Application</Link>
            </li>
          </ul>
        </nav>

        <Switch>
          <Route path="/status/:id" exact component={CheckStatus}>
          </Route>
          <Route path="/action/:id" exact component={AdminAction}>
          </Route>
          <Route path="/">
            <Application></Application>
          </Route>
        </Switch>
      </div>
    </Router>
  );
}

export default App;
