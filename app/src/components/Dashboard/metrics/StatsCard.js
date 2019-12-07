import React from "react";
import clsx from "clsx";
import PropTypes from "prop-types";
import { makeStyles } from "@material-ui/styles";
import { Card, CardContent, Grid, Typography, Avatar } from "@material-ui/core";
import HourglassEmptyIcon from "@material-ui/icons/HourglassEmpty";
import NotInterestedIcon from "@material-ui/icons/NotInterested";
import CheckCircleIcon from "@material-ui/icons/CheckCircle";

import AlarmOnIcon from "@material-ui/icons/AlarmOn";

const useStyles = makeStyles(theme => ({
  root: {
    height: "100%"
  },
  content: {
    alignItems: "center",
    display: "flex"
  },
  title: {
    fontWeight: 700
  },
  icon: {
    height: 32,
    width: 32
  },
  difference: {
    marginTop: theme.spacing(2),
    display: "flex",
    alignItems: "center"
  },
  differenceIcon: {
    color: theme.palette.success.dark
  },
  differenceValue: {
    color: theme.palette.success.dark,
    marginRight: theme.spacing(1)
  }
}));

const StatsCard = props => {
  const { className, ...rest } = props;

  const classes = useStyles();

  let icon;
  if (props.type === "Approved") {
    icon = <CheckCircleIcon className={classes.icon} />;
  } else if (props.type === "Banned") {
    icon = <NotInterestedIcon className={classes.icon} />;
  } else if (props.type === "Pending") {
    icon = <HourglassEmptyIcon className={classes.icon} />;
  } else if (props.type === "ResponseTime") {
    icon = <AlarmOnIcon className={classes.icon}></AlarmOnIcon>;
  }
  return (
    <Card {...rest} className={clsx(classes.root, className)}>
      <CardContent>
        <Grid container justify="space-between">
          <Grid item>
            <Typography
              className={classes.title}
              color="textSecondary"
              gutterBottom
              variant="body2"
            >
              {props.title}
            </Typography>
            <Typography variant="h3">{props.value}</Typography>
          </Grid>
          <Grid item>{icon}</Grid>
        </Grid>
      </CardContent>
    </Card>
  );
};

StatsCard.propTypes = {
  className: PropTypes.string
};

export default StatsCard;
