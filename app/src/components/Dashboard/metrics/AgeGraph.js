import React from "react";
import clsx from "clsx";
import PropTypes from "prop-types";
import { makeStyles } from "@material-ui/core/styles";
import { Card, CardHeader, CardContent, Divider } from "@material-ui/core";
import {
  ResponsiveContainer,
  Legend,
  Tooltip,
  RadialBarChart,
  RadialBar
} from "recharts";

const useStyles = makeStyles(theme => ({
  root: {
    height: "30vh"
  },
  chartContainer: {
    position: "relative",
    height: "300px"
  },
  stats: {
    marginTop: theme.spacing(2),
    display: "flex",
    justifyContent: "center"
  },
  device: {
    textAlign: "center",
    padding: theme.spacing(1)
  },
  deviceIcon: {
    color: theme.palette.icon
  }
}));

const AgeGraph = props => {
  const { className, ...rest } = props;

  const classes = useStyles();

  let ageGroupData = [
    {
      name: "0-15",
      value: props.ageGroup1Count,
      fill: "#8884d8"
    },
    {
      name: "15-30",
      value: props.ageGroup2Count,
      fill: "#83a6ed"
    },
    {
      name: "30-45",
      value: props.ageGroup3Count,
      fill: "#8dd1e1"
    },
    {
      name: "45+",
      value: props.ageGroup4Count,
      fill: "#82ca9d"
    }
  ];

  return (
    <Card {...rest} className={clsx(classes.root, className)}>
      <CardHeader title="Age Distribution of Players" />
      <Divider />
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <RadialBarChart
            innerRadius="10%"
            outerRadius="80%"
            data={ageGroupData}
            startAngle={180}
            endAngle={0}
          >
            <RadialBar
              minAngle={15}
              background
              clockWise={true}
              dataKey="value"
            />
            <Legend
              iconSize={10}
              layout="vertical"
              verticalAlign="top"
              align="right"
            />
            <Tooltip />
          </RadialBarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
};

AgeGraph.propTypes = {
  className: PropTypes.string
};

export default AgeGraph;
