package ch.epfl.prifiproxy.viewmodel;

import android.app.Application;
import android.arch.lifecycle.AndroidViewModel;
import android.arch.lifecycle.LiveData;
import android.support.annotation.NonNull;

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.repository.ConfigurationGroupRepository;
import ch.epfl.prifiproxy.repository.ConfigurationRepository;

public class ConfigurationAddEdditViewModel extends AndroidViewModel {
    private ConfigurationRepository configurationRepository;
    private ConfigurationGroupRepository groupRepository;
    private LiveData<Configuration> configuration;
    private LiveData<ConfigurationGroup> group;
    private int groupId;

    public ConfigurationAddEdditViewModel(@NonNull Application application) {
        super(application);
        configurationRepository = ConfigurationRepository.getInstance(application);
        groupRepository = ConfigurationGroupRepository.getInstance(application);
    }

    public void init(int groupId, int configurationId) {
        this.groupId = groupId;
        configuration = configurationRepository.getConfiguration(configurationId);
        group = groupRepository.getGroup(groupId);
    }

    public int getGroupId() {
        return groupId;
    }

    public LiveData<ConfigurationGroup> getGroup() {
        return group;
    }

    public LiveData<Configuration> getConfiguration() {
        return configuration;
    }

    public void insert(Configuration configuration) {
        configurationRepository.insert(configuration);
    }

    public void update(Configuration configuration) {
        configurationRepository.update(configuration);
    }

    public void delete(Configuration configuration) {
        configurationRepository.delete(configuration);
    }

    public void insertOrUpdate(Configuration configuration) {
        if (configuration.getId() == 0) {
            insert(configuration);
        } else {
            update(configuration);
        }
    }
}
