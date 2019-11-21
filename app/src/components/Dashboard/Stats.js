/* eslint-disable no-script-url */

import React from "react";
import { makeStyles } from "@material-ui/core/styles";
import moment from "moment";
import Typography from "@material-ui/core/Typography";
import Title from "./Title";
import i18next from "i18next";

const useStyles = makeStyles({
  depositContext: {
    flex: 1
  }
});

const getAverageResponseTimeInHours = fulfilledRequests => {
  let total = 0;
  fulfilledRequests.forEach(request => {
    let start = moment.parseZone(request["timestamp"]);
    let end = moment.parseZone(request["processedTimestamp"]);
    let duration = moment.duration(end.diff(start));
    total += duration.asHours();
  });
  // Avoid devide by zero
  if (fulfilledRequests.length) {
    return 0;
  }
  return total / fulfilledRequests.length;
};

export default function Stats(props) {
  const classes = useStyles();
  return (
    <React.Fragment>
      <Title>Stats</Title>
      <Typography component="p" variant="h4">
        {props.data.pendingRequests.length}
      </Typography>
      <Typography color="textSecondary" className={classes.depositContext}>
        {i18next.t("Dashboard.Stats.Pending")}
      </Typography>
      <Typography component="p" variant="h4">
        {props.data.fulfilledRequests.length}
      </Typography>
      <Typography color="textSecondary" className={classes.depositContext}>
        {i18next.t("Dashboard.Stats.Fulfilled")}
      </Typography>
      <Typography component="p" variant="h4">
        {getAverageResponseTimeInHours(props.data.fulfilledRequests).toFixed(1)}{" "}
        Hours
      </Typography>
      <Typography color="textSecondary" className={classes.depositContext}>
        {i18next.t("Dashboard.Stats.ResponseTime")}
      </Typography>
    </React.Fragment>
  );
}
