package ch.epfl.prifiproxy.repository;

import android.app.Application;
import android.arch.lifecycle.LiveData;
import android.os.AsyncTask;

import java.util.List;

import ch.epfl.prifiproxy.persistence.AppDatabase;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class ConfigurationRepository {
    private ConfigurationDao configurationDao;
    private LiveData<List<ConfigurationGroup>> allGroups;
    private LiveData<List<Configuration>> allConfigurations;

    public ConfigurationRepository(Application application) {
        AppDatabase db = AppDatabase.getDatabase(application);
        configurationDao = db.configurationDao();
        allGroups = configurationDao.getAllConfigurationGroups();
        allConfigurations = configurationDao.getAllConfigurations();
    }

    public LiveData<List<ConfigurationGroup>> getAllGroups() {
        return allGroups;
    }

    public LiveData<List<Configuration>> getAllConfigurations() {
        return allConfigurations;
    }

    public void insert(ConfigurationGroup group) {
        new insertAsyncTask(configurationDao).execute(group);
    }

    private static class insertAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationDao dao;

        insertAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final ConfigurationGroup... configurationGroups) {
            dao.insertConfigurationGroups(configurationGroups);
            return null;
        }
    }
}
