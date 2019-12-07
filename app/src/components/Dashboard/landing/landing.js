import React from "react";
import clsx from "clsx";
import { withStyles } from "@material-ui/core/styles";
import Container from "@material-ui/core/Container";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import Chart from "./Chart";
import Table from "./Table";
import RequestsService from "../../../service/RequestsService";
import { withRouter } from "react-router-dom";

const useStyles = theme => ({
  container: {
    paddingTop: theme.spacing(4),
    paddingBottom: theme.spacing(4)
  },
  paper: {
    padding: theme.spacing(2),
    display: "flex",
    overflow: "auto",
    flexDirection: "column"
  },
  fixedHeight: {
    height: 240
  }
});

class Landing extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      requests: null
    };
  }

  componentDidMount() {
    // Check localstorage for token, if null or invalid, redirects to login page
    let token = JSON.parse(localStorage.getItem("token"));
    if (!token) {
      this.props.history.push("/login");
      return;
    }

    // such request will go through the auth middleware to check the validity of token implicitly
    // So if status code returned is 401 we could redirect user to login
    let config = {
      headers: {
        Authorization: `Bearer ${token.value}`
      }
    };
    this.setState({ auth_header: config });
    RequestsService.getAllRequests(config)
      .then(res => {
        if (res.status === 200) {
          this.setState({
            requests: res.data.requests
          });
        }
      })
      .catch(error => {
        // direct unauthenticated to login
        // In the case that localstorage has expired / invalid token, clear that up
        localStorage.clear();
        this.props.history.push("/login");
        return;
      });
  }

  handleChangeRequestStatus = requestIDandUpdatedRequest => {
    let requestID = requestIDandUpdatedRequest.requestID;
    let updatedRequest = requestIDandUpdatedRequest.request;
    let requests = this.state.requests;
    for (var i = 0; i < requests.length; i++) {
      if (requests[i]._id === requestID) {
        requests[i] = updatedRequest;
      }
    }
    this.setState({ requests });
  };
  render() {
    if (this.state.requests == null) {
      return <div>Loading...</div>;
    }
    const { classes } = this.props;
    const fixedHeightPaper = clsx(classes.paper, classes.fixedHeight);
    return (
      <Container maxWidth="lg" className={classes.container}>
        <Grid container spacing={3}>
          {/* Chart */}
          <Grid item xs={12} md={12} lg={12}>
            <Paper className={fixedHeightPaper}>
              <Chart requests={this.state.requests} />
            </Paper>
          </Grid>
          {/* Whitelist request table-view */}
          <Grid item xs={12}>
            <Paper className={classes.paper}>
              <Table
                requests={this.state.requests}
                config={this.state.auth_header}
                handleChangeRequestStatus={requestID =>
                  this.handleChangeRequestStatus(requestID)
                }
              />
            </Paper>
          </Grid>
        </Grid>
      </Container>
    );
  }
}

export default withRouter(withStyles(useStyles)(Landing));
