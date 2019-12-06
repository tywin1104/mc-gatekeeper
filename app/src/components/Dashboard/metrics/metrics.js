import React from "react";
import { withStyles } from "@material-ui/core/styles";
import Container from "@material-ui/core/Container";
import Grid from "@material-ui/core/Grid";
import SnackbarContent from "@material-ui/core/SnackbarContent";
import RequestsService from "../../../service/RequestsService";
import PropTypes from "prop-types";
import clsx from "clsx";
import CheckCircleIcon from "@material-ui/icons/CheckCircle";
import { amber, green } from "@material-ui/core/colors";
import WarningIcon from "@material-ui/icons/Warning";
import { makeStyles } from "@material-ui/core/styles";
import StatusGraph from "./StatusGraph";
import AgeGraph from "./AgeGraph";
import PerformanceChart from "./PerformanceChart";
import GenderGraph from "./GenderGraph";
import StatsCard from "./StatsCard";

const useStyles = theme => ({
  root: {
    marginTop: "5%",
    flexGrow: 1
  }
});

const variantIcon = {
  success: CheckCircleIcon,
  warning: WarningIcon
};

const useStyles1 = makeStyles(theme => ({
  success: {
    backgroundColor: green[600]
  },
  warning: {
    backgroundColor: amber[700]
  },
  icon: {
    fontSize: 20
  },
  iconVariant: {
    opacity: 0.9,
    marginRight: theme.spacing(1)
  },
  message: {
    display: "flex",
    alignItems: "center"
  }
}));

function MySnackbarContentWrapper(props) {
  const classes = useStyles1();
  const { className, message, onClose, variant, ...other } = props;
  const Icon = variantIcon[variant];

  return (
    <SnackbarContent
      className={clsx(classes[variant], className)}
      aria-describedby="client-snackbar"
      message={
        <span id="client-snackbar" className={classes.message}>
          <Icon className={clsx(classes.icon, classes.iconVariant)} />
          {message}
        </span>
      }
      {...other}
    />
  );
}

MySnackbarContentWrapper.propTypes = {
  className: PropTypes.string,
  message: PropTypes.string,
  onClose: PropTypes.func,
  variant: PropTypes.oneOf(["success", "warning"]).isRequired
};

class Metrics extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      source: RequestsService.getStatsEventSource(),
      stats: null
    };
  }
  componentDidMount() {
    const { source } = this.state;
    source.addEventListener("closedConnection", e => this.source.close());
    // Update the stats once data arrived from the sever via SSE
    source.addEventListener("message", message => {
      this.updateStats(message.data);
    });
  }

  componentWillUnmount() {
    this.state.source.close();
  }

  updateStats = data => {
    this.setState({ stats: JSON.parse(data) });
  };

  _getAverageResponseTime = () => {
    let value = this.state.stats.averageResponseTimeInMinutes;
    if (value === 0) {
      return "N/A";
    } else {
      return `${value} Minutes`;
    }
  };

  _getOvertimeWarningMsg = () => {
    if (this.state.stats != null && this.state.stats.aggregateStats != null) {
      let count = this.state.stats.aggregateStats.overtimeCount;
      if (count > 0) {
        return `There are ${count} unhandled pending requests for more than 24 hours.`;
      } else {
        return "There is no pending request awaiting for more than 24 hours. Nice Job!";
      }
    }
    return "";
  };

  _getSnackBarStyle = () => {
    if (this.state.stats != null && this.state.stats.aggregateStats != null) {
      let count = this.state.stats.aggregateStats.overtimeCount;
      if (count > 0) {
        return "warning";
      }
    }
    return "success";
  };

  getGraphs = () => {
    if (this.state.stats == null || this.state.stats.approved === 0) {
      return <p>Insufficient data to generate graph</p>;
    }
    const section = {
      height: "100%",
      paddingTop: 5,
      backgroundColor: "#fff"
    };
    // Display gender and age graphs if there are approved records
    return (
      <React.Fragment>
        <Grid item lg={6} md={6} xl={6} xs={12}>
          <div style={section}>
            <GenderGraph
              maleCount={this.state.stats.maleCount}
              femaleCount={this.state.stats.femaleCount}
              otherGenderCount={this.state.stats.otherGenderCount}
            ></GenderGraph>
          </div>
        </Grid>
        <Grid item lg={6} md={6} xl={6} xs={12}>
          <div style={section}>
            <StatusGraph
              pending={this.state.stats.pending}
              approved={this.state.stats.approved}
              denied={this.state.stats.denied}
            ></StatusGraph>
          </div>
        </Grid>
        <Grid item lg={6} md={6} xl={6} xs={12}>
          <div style={section}>
            <AgeGraph
              ageGroup1Count={this.state.stats.ageGroup1Count}
              ageGroup2Count={this.state.stats.ageGroup2Count}
              ageGroup3Count={this.state.stats.ageGroup3Count}
              ageGroup4Count={this.state.stats.ageGroup4Count}
            ></AgeGraph>
          </div>
        </Grid>
        <Grid item lg={6} md={6} xl={6} xs={12}>
          <div style={section}>
            <PerformanceChart
              aggregateStats={this.state.stats.aggregateStats}
            ></PerformanceChart>
          </div>
        </Grid>
      </React.Fragment>
    );
  };

  _getStatsCard = (title, data, type) => {
    return (
      <Grid item xs={12} sm={3}>
        <StatsCard title={title} value={data} type={type}></StatsCard>
      </Grid>
    );
  };

  getStatsCards = () => {
    if (this.state.stats == null) {
      return;
    }
    return (
      <React.Fragment>
        {this._getStatsCard(
          "Pending Requests Count",
          this.state.stats.pending,
          "Pending"
        )}
        {this._getStatsCard(
          "Approved Requests Count",
          this.state.stats.approved,
          "Approved"
        )}
        {this._getStatsCard(
          "Denied Requests Count",
          this.state.stats.denied,
          "Denied"
        )}
        {this._getStatsCard(
          "Average Response Time",
          this._getAverageResponseTime(),
          "ResponseTime"
        )}
      </React.Fragment>
    );
  };

  getNotification = () => {
    return (
      <Grid item xs={12} sm={12} height="100%">
        <MySnackbarContentWrapper
          variant={this._getSnackBarStyle()}
          message={this._getOvertimeWarningMsg()}
        />
      </Grid>
    );
  };
  render() {
    const { classes } = this.props;
    return (
      <Container maxWidth="lg" className={classes.root}>
        <Grid
          container
          direction="row"
          justify="flex-start"
          alignItems="flex-start"
          spacing={3}
        >
          {this.getNotification()}
          {this.getStatsCards()}
          {this.getGraphs()}
        </Grid>
      </Container>
    );
  }
}

export default withStyles(useStyles)(Metrics);
