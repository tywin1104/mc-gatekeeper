import React from "react";
import { Doughnut } from "react-chartjs-2";
import clsx from "clsx";
import PropTypes from "prop-types";
import { makeStyles } from "@material-ui/core/styles";
import { useTheme } from "@material-ui/core/styles";
import {
  Card,
  CardHeader,
  CardContent,
  Divider,
  Typography
} from "@material-ui/core";
import PregnantWomanIcon from "@material-ui/icons/PregnantWoman";
import AccessibilityIcon from "@material-ui/icons/Accessibility";
import HelpIcon from "@material-ui/icons/Help";

const useStyles = makeStyles(theme => ({
  root: {
    height: "40vh"
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

const GenderGraph = props => {
  const { className, ...rest } = props;

  const classes = useStyles();

  const theme = useTheme();

  const data = {
    datasets: [
      {
        data: [props.maleCount, props.femaleCount, props.otherGenderCount],
        backgroundColor: [
          theme.palette.primary.main,
          theme.palette.error.main,
          theme.palette.warning.main
        ],
        borderWidth: 8,
        borderColor: theme.palette.white,
        hoverBorderColor: theme.palette.white
      }
    ],
    labels: ["Male", "Female", "Others"]
  };

  const options = {
    legend: {
      display: false
    },
    responsive: true,
    maintainAspectRatio: false,
    animation: true,
    cutoutPercentage: 80,
    layout: { padding: 0 },
    tooltips: {
      enabled: true,
      mode: "index",
      intersect: false,
      borderWidth: 1,
      borderColor: theme.palette.divider,
      backgroundColor: theme.palette.white,
      titleFontColor: theme.palette.text.primary,
      bodyFontColor: theme.palette.text.secondary,
      footerFontColor: theme.palette.text.secondary
    }
  };

  const devices = [
    {
      title: "Male",
      value: props.maleCount,
      icon: <AccessibilityIcon />,
      color: theme.palette.primary.main
    },
    {
      title: "Female",
      value: props.femaleCount,
      icon: <PregnantWomanIcon />,
      color: theme.palette.error.main
    },
    {
      title: "Others",
      value: props.otherGenderCount,
      icon: <HelpIcon />,
      color: theme.palette.warning.main
    }
  ];

  return (
    <Card {...rest} className={clsx(classes.root, className)}>
      <CardHeader title="Gender Distribution of Players" />
      <Divider />
      <CardContent>
        <div>
          <Doughnut data={data} options={options} />
        </div>
        <div className={classes.stats}>
          {devices.map(device => (
            <div className={classes.device} key={device.title}>
              <span className={classes.deviceIcon}>{device.icon}</span>
              <Typography variant="body1">{device.title}</Typography>
              <Typography style={{ color: device.color }} variant="h3">
                {device.value}
              </Typography>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};

GenderGraph.propTypes = {
  className: PropTypes.string
};

export default GenderGraph;
