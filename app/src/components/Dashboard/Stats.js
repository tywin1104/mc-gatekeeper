/* eslint-disable no-script-url */

import React from "react";

import Typography from "@material-ui/core/Typography";
import Title from "./Title";
import i18next from "i18next";
import RequestsService from "../../service/RequestsService";

class Stats extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      source: RequestsService.getStatsEventSource(),
      stats: {}
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

  updateStats = data => {
    this.setState({ stats: JSON.parse(data) });
  };

  render() {
    return (
      <React.Fragment>
        <Title>Stats</Title>
        <Typography component="p" variant="h4">
          {this.state.stats.pending}
        </Typography>
        <Typography color="textSecondary">
          {i18next.t("Dashboard.Stats.Pending")}
        </Typography>
        <Typography component="p" variant="h4">
          {this.state.stats.approved}
        </Typography>
        <Typography color="textSecondary">
          {i18next.t("Dashboard.Stats.Approved")}
        </Typography>
        <Typography component="p" variant="h4">
          {this.state.stats.denied}
        </Typography>
        <Typography color="textSecondary">
          {i18next.t("Dashboard.Stats.Denied")}
        </Typography>
        <Typography component="p" variant="h4">
          {this.state.stats.averageResponseTimeInMinutes} Minutes
        </Typography>
        <Typography color="textSecondary">
          {i18next.t("Dashboard.Stats.ResponseTime")}
        </Typography>
      </React.Fragment>
    );
  }
}

export default Stats;
