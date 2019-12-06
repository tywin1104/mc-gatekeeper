import React from "react";
import clsx from "clsx";
import PropTypes from "prop-types";
import { makeStyles } from "@material-ui/core/styles";
import { useTheme } from "@material-ui/core/styles";
import { Card, CardHeader, CardContent, Divider } from "@material-ui/core";
import {
  ResponsiveContainer,
  Legend,
  Tooltip,
  BarChart,
  CartesianGrid,
  XAxis,
  YAxis,
  Bar
} from "recharts";

const useStyles = makeStyles(theme => ({
  root: {
    height: "40vh"
  }
}));

const StatusGraph = props => {
  const { className, ...rest } = props;

  const classes = useStyles();

  const data = [
    {
      name: "Pending",
      count: props.pending
    },
    {
      name: "Approved",
      count: props.approved
    },
    {
      name: "Denied",
      count: props.denied
    }
  ];

  return (
    <Card {...rest} className={clsx(classes.root, className)}>
      <CardHeader title="Status Distribution Graph" />
      <Divider />
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="count" fill="#8884d8" />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
};

StatusGraph.propTypes = {
  className: PropTypes.string
};

export default StatusGraph;
