/* eslint-disable no-script-url */

import React from 'react';
import { makeStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import Title from './Title';

const useStyles = makeStyles({
  depositContext: {
    flex: 1,
  },
});

export default function Stats(props) {
  const classes = useStyles();
  return (
    <React.Fragment>
      <Title>Stats</Title>
      <Typography component="p" variant="h4">
          {props.pending}
      </Typography>
      <Typography color="textSecondary" className={classes.depositContext}>
          Pending requests
      </Typography>
      <Typography component="p" variant="h4">
        {props.fulfilled}
      </Typography>
      <Typography color="textSecondary" className={classes.depositContext}>
          Fulfulled requests
      </Typography>
    </React.Fragment>
  );
}