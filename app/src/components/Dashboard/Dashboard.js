import React from 'react';
import clsx from 'clsx';
import { withStyles } from "@material-ui/core/styles";
import CssBaseline from '@material-ui/core/CssBaseline';
import Drawer from '@material-ui/core/Drawer';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import List from '@material-ui/core/List';
import Typography from '@material-ui/core/Typography';
import Divider from '@material-ui/core/Divider';
import IconButton from '@material-ui/core/IconButton';
import Container from '@material-ui/core/Container';
import Grid from '@material-ui/core/Grid';
import Paper from '@material-ui/core/Paper';
import Link from '@material-ui/core/Link';
import MenuIcon from '@material-ui/icons/Menu';
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft';
import { mainListItems } from './listItems';
import Chart from './Chart';
import Stats from './Stats';
import Table from './Table';
import RequestsService from '../../service/RequestsService';
import i18next from "i18next";



const drawerWidth = 240;
const useStyles = theme => ({
  root: {
    display: 'flex',
  },
  toolbar: {
    paddingRight: 24, // keep right padding when drawer closed
  },
  toolbarIcon: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    padding: '0 8px',
    ...theme.mixins.toolbar,
  },
  appBar: {
    zIndex: theme.zIndex.drawer + 1,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
  },
  appBarShift: {
    marginLeft: drawerWidth,
    width: `calc(100% - ${drawerWidth}px)`,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  },
  menuButton: {
    marginRight: 36,
  },
  menuButtonHidden: {
    display: 'none',
  },
  title: {
    flexGrow: 1,
  },
  drawerPaper: {
    position: 'relative',
    whiteSpace: 'nowrap',
    width: drawerWidth,
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  },
  drawerPaperClose: {
    overflowX: 'hidden',
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
    width: theme.spacing(7),
    [theme.breakpoints.up('sm')]: {
      width: theme.spacing(9),
    },
  },
  appBarSpacer: theme.mixins.toolbar,
  content: {
    flexGrow: 1,
    height: '100vh',
    overflow: 'auto',
  },
  container: {
    paddingTop: theme.spacing(4),
    paddingBottom: theme.spacing(4),
  },
  paper: {
    padding: theme.spacing(2),
    display: 'flex',
    overflow: 'auto',
    flexDirection: 'column',
  },
  fixedHeight: {
    height: 240,
  },
})

class Dashboard extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      open: true,
      requests: [],
      auth_header : {}
    };
  }

  componentDidMount() {
    // Check localstorage for token, if null or invalid, redirects to login page
    let token = JSON.parse(localStorage.getItem("token"));
    if(!token) {
      this.props.history.push("/login")
      return
    }

    // such request will go through the auth middleware to check the validity of token implicitly
    // So if status code returned is 401 we could redirect user to login
    let config = {
      headers: {
        Authorization: `Bearer ${token.value}`,
      }
    }
    this.setState({auth_header: config})
    RequestsService.getAllRequests(config)
    .then(res => {
      if (res.status === 200) {
          this.setState({
            requests: res.data.requests
          })
      }})
    .catch(error => {
      // direct unauthenticated to login
      // In the case that localstorage has expired / invalid token, clear that up
      localStorage.clear();
      this.props.history.push("/login")
      return
    });
  }

  handleDrawerOpen = () => {
    this.setState({
      open: true
    })
  };
  handleDrawerClose = () => {
    this.setState({
      open: false
    })
  };

  handleChangeRequestStatus = (requestID, newStatus) => {
    let requests = this.state.requests
    requests.forEach((request)=> {
      if(request._id === requestID) {
          request.status = newStatus
      }
    })
    this.setState({requests})
  }

  render() {
    if(!this.state.requests) {
      return <div>Loading...</div>
    }
    let pendingRequests = this.state.requests.filter(request => {
        return request.status === "Pending"
    })
    let fulfilledRequests = this.state.requests.filter(request => {
      return request.status === "Approved" || request.status === "Denied"
    })
    let approvedRequests = this.state.requests.filter(request => {
      return request.status === "Approved"
    })
    let statsSource = {pendingRequests, fulfilledRequests, approvedRequests}
    const { classes } = this.props
    const fixedHeightPaper = clsx(classes.paper, classes.fixedHeight);
    return (
      <div className={classes.root}>
      <CssBaseline />
      <AppBar position="absolute" className={clsx(classes.appBar, this.state.open && classes.appBarShift)}>
        <Toolbar className={classes.toolbar}>
          <IconButton
            edge="start"
            color="inherit"
            aria-label="open drawer"
            onClick={this.handleDrawerOpen}
            className={clsx(classes.menuButton, this.state.open && classes.menuButtonHidden)}
          >
            <MenuIcon />
          </IconButton>
          <Typography component="h1" variant="h6" color="inherit" noWrap className={classes.title}>
            {i18next.t('Dashboard.Title')}
          </Typography>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        classes={{
          paper: clsx(classes.drawerPaper, !this.state.open && classes.drawerPaperClose),
        }}
        open={this.state.open}
      >
        <div className={classes.toolbarIcon}>
          <IconButton onClick={this.handleDrawerClose}>
            <ChevronLeftIcon />
          </IconButton>
        </div>
        <Divider />
        <List>{mainListItems}</List>
      </Drawer>
      <main className={classes.content}>
        <div className={classes.appBarSpacer} />
        <Container maxWidth="lg" className={classes.container}>
          <Grid container spacing={3}>
            {/* Chart */}
            <Grid item xs={12} md={8} lg={9}>
              <Paper className={fixedHeightPaper}>
                <Chart requests={this.state.requests}/>
              </Paper>
            </Grid>
            {/* Stats */}
            <Grid item xs={12} md={4} lg={3}>
              <Paper className={fixedHeightPaper}>
                <Stats data={statsSource} />
              </Paper>
            </Grid>
            {/* Whitelist request table-view */}
            <Grid item xs={12}>
              <Paper className={classes.paper}>
                <Table
                    requests={this.state.requests} config={this.state.auth_header}
                    handleChangeRequestStatus={(requestID) => this.handleChangeRequestStatus(requestID)}
                />
              </Paper>
            </Grid>
          </Grid>
        </Container>
        <Typography variant="body2" color="textSecondary" align="center">
        {'Copyright © '}
        <Link color="inherit">
          Your Website
        </Link>{' '}
        {new Date().getFullYear()}
        {'.'}
      </Typography>
      </main>
    </div>
    )
  }
}

export default withStyles(useStyles)(Dashboard);