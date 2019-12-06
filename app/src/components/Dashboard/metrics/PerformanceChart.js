import React, { useState } from "react";
import clsx from "clsx";
import PropTypes from "prop-types";
import { makeStyles } from "@material-ui/styles";
import {
  Card,
  CardHeader,
  CardContent,
  Divider,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText
} from "@material-ui/core";

const _createData = (name, count, averageResponseTimeInMinutes) => {
  return { name, count, averageResponseTimeInMinutes };
};

const _getTableRows = props => {
  let rows = [];
  if (props.aggregateStats != null) {
    let adminPerformance = props.aggregateStats.adminPerformance;
    const admins = Object.keys(props.aggregateStats.adminPerformance);
    admins.forEach(admin => {
      rows.push(
        _createData(
          admin,
          adminPerformance[admin].totalHandled,
          adminPerformance[admin].averageResponseTimeInMinutes
        )
      );
    });
  }
  return rows;
};

const useStyles = makeStyles(() => ({
  root: {
    height: "30vh"
  },
  content: {
    padding: 0
  },
  image: {
    height: 48,
    width: 48
  },
  actions: {
    justifyContent: "flex-end"
  }
}));

const PerformanceChart = props => {
  const { className, ...rest } = props;

  const classes = useStyles();

  let data = _getTableRows(props);
  const admins = data;

  return (
    <Card {...rest} className={clsx(classes.root, className)}>
      <CardHeader
        subtitle={`${admins.length} in total`}
        title="Server Admins Performance Overview"
      />
      <Divider />
      <CardContent className={classes.content}>
        <List>
          {admins.map((admin, i) => (
            <ListItem divider={i < admins.length - 1}>
              <ListItemAvatar>
                <img
                  alt="Avatar"
                  className={classes.image}
                  src={`https://ui-avatars.com/api/?name=${admin.name}`}
                />
              </ListItemAvatar>
              <ListItemText
                primary={admin.name}
                secondary={`Handled ${admin.count} applications in total `}
              />
              <ListItemText
                secondary={`Average response time is ${admin.averageResponseTimeInMinutes.toFixed(
                  0
                )} minutes`}
              />
            </ListItem>
          ))}
        </List>
      </CardContent>
      <Divider />
    </Card>
  );
};

PerformanceChart.propTypes = {
  className: PropTypes.string
};

export default PerformanceChart;
