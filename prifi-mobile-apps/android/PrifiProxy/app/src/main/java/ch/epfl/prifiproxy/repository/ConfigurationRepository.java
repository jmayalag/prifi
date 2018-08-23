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
        new InsertAsyncTask(configurationDao).execute(group);
    }

    public void updateGroups(List<ConfigurationGroup> groups) {
        new UpdateAsyncTask(configurationDao)
                .execute(groups.toArray(new ConfigurationGroup[groups.size()]));
    }

    private static class InsertAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationDao dao;

        InsertAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final ConfigurationGroup... configurationGroups) {
            dao.insertConfigurationGroups(configurationGroups);
            return null;
        }
    }

    private static class UpdateAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationDao dao;

        UpdateAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(ConfigurationGroup... groups) {
            dao.updateConfigurationGroups(groups);
            return null;
        }
    }
}
